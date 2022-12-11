/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	omerv1 "omer.io/namespacelabel/api/v1"
)

// NamespaceLabelReconciler reconciles a NamespaceLabel object
type NamespaceLabelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=omer.omer.io,resources=namespacelabels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=omer.omer.io,resources=namespacelabels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=omer.omer.io,resources=namespacelabels/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceLabel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile

// the finalizer label for namespacelabel object
const nsLabelFinalizer = "namespacelabel.omer.io/finalizer"

func isNsLabelInDeletionState(namespaceLabel omerv1.NamespaceLabel) bool {
	return !(namespaceLabel.ObjectMeta.DeletionTimestamp.IsZero())
}

func isLabelKeyExistInLabels(labels map[string]string, key string) bool {
	_, isExist := labels[key]
	return isExist
}

func (r *NamespaceLabelReconciler) updateSyncNamespaceLabel(ctx context.Context, key string, value string, namespaceLabel *omerv1.NamespaceLabel) error {
	//if there is no labels in the namespaceLabel status - create!
	if (*namespaceLabel).Status.SyncLabels == nil {
		fmt.Println("there is no labels in the namespaceLabel status")
		(*namespaceLabel).Status.SyncLabels = make(map[string]string)
	}
	//update the nslabel sync status
	(*namespaceLabel).Status.SyncLabels[key] = value

	return nil
}

func (r *NamespaceLabelReconciler) updateNamespaceLabels(ctx context.Context, key string, value string, namespace *v1.Namespace) error {
	//if there is no labels in the namespace - create!
	if (*namespace).Labels == nil {
		(*namespace).Labels = make(map[string]string)
	}
	//update the label
	(*namespace).Labels[key] = value

	return nil
}

func (r *NamespaceLabelReconciler) cleanupNamespaceLabelFix(ctx context.Context, namespaceLabel omerv1.NamespaceLabel, nsLabelFinalizer string) error {
	//get the namespace for sync to nslabel
	var namespace v1.Namespace
	namespacedName := types.NamespacedName{Name: namespaceLabel.Namespace}
	if err := r.Get(ctx, namespacedName, &namespace); err != nil {
		fmt.Println("unable to fetch namespace")
		return client.IgnoreNotFound(err)
	}

	//delete all sync labels from ns
	isChangeNeededInNamespace := false
	for key, _ := range namespaceLabel.Status.SyncLabels {
		if isLabelKeyExistInLabels(namespace.ObjectMeta.Labels, key) {
			isChangeNeededInNamespace = true
			delete(namespace.ObjectMeta.Labels, key)
		}
	}

	//update the ns
	if isChangeNeededInNamespace {
		if err := r.Update(ctx, &namespace); err != nil {
			fmt.Println(err, "unable to update namespace")
			return err
		}
	}
	//remove the finalizer
	controllerutil.RemoveFinalizer(&namespaceLabel, nsLabelFinalizer)
	if err := r.Update(ctx, &namespaceLabel); err != nil {
		return err
	}

	return nil
}

func (r *NamespaceLabelReconciler) synchronizedNamespaceLabelToNamespaceFix(ctx context.Context, namespaceLabel omerv1.NamespaceLabel) error {
	//get the namespace for sync to nslabel
	var namespace v1.Namespace
	namespacedName := types.NamespacedName{Name: namespaceLabel.Namespace}
	if err := r.Get(ctx, namespacedName, &namespace); err != nil {
		fmt.Println("unable to fetch namespace")
		return client.IgnoreNotFound(err)
	}

	//stage 1:the label is in sync and not in spec->the label deleted and should delete from the nslabel sync and ns
	//first we will delete the label from the ns(stage 1.1), than check if it already deleted from the ns - we will delete from nslabel(stage 1.2)
	isChangeNeededInNamespace := false
	isChangeNeededInNslabelSync := false
	for key, _ := range namespaceLabel.Status.SyncLabels {
		if !(isLabelKeyExistInLabels(namespaceLabel.Spec.Labels, key)) {
			if isLabelKeyExistInLabels(namespace.ObjectMeta.Labels, key) {
				fmt.Println("HARA 1")
				isChangeNeededInNamespace = true
				delete(namespace.ObjectMeta.Labels, key)
			} else if isLabelKeyExistInLabels(namespaceLabel.Status.SyncLabels, key) {
				fmt.Println("HARA 2")
				isChangeNeededInNslabelSync = true
				delete(namespaceLabel.Status.SyncLabels, key)
			}
		}
	}
	if isChangeNeededInNamespace {
		fmt.Println("stage 1.1:->")
		if err := r.Update(ctx, &namespace); err != nil {
			fmt.Println(err, "unable to update namespace")
			return err
		}
	}
	if isChangeNeededInNslabelSync {
		fmt.Println("stage 1.2:->")
		if err := r.Status().Update(ctx, &namespaceLabel); err != nil {
			fmt.Println(err, "unable to update status of namespaceLabel")
			return err
		}
		return nil
	}

	//stage 2:running on all the labels in the spec of the nslabel object
	//stage 2.1: the label is in the namespace and not in the sync - result: delete from nslabel spec
	//stage 2.2: the label is in the namespace and in the sync - result: update the sync
	//stage 2.3: the label is in the not in the namespace and also not in the sync - result: update the sync
	isChangeNeededInNsLabelSpec := false
	isChangeNeededInNsLabelSync := false
	for key, value := range namespaceLabel.Spec.Labels {
		fmt.Println("key:", key, "value:", value)
		if isLabelKeyExistInLabels(namespace.ObjectMeta.Labels, key) && !(isLabelKeyExistInLabels(namespaceLabel.Status.SyncLabels, key)) {
			isChangeNeededInNsLabelSpec = true
			delete(namespaceLabel.Spec.Labels, key)
		} else if isLabelKeyExistInLabels(namespace.ObjectMeta.Labels, key) && isLabelKeyExistInLabels(namespaceLabel.Status.SyncLabels, key) && (value != namespaceLabel.Status.SyncLabels[key]) && (!isChangeNeededInNsLabelSpec) {
			isChangeNeededInNsLabelSync = true
			if err := r.updateSyncNamespaceLabel(ctx, key, value, &namespaceLabel); err != nil {
				fmt.Println(err, "unable to update sync namespace label")
				return err
			}
		} else if (!isLabelKeyExistInLabels(namespace.ObjectMeta.Labels, key)) && (!isLabelKeyExistInLabels(namespaceLabel.Status.SyncLabels, key)) && (!isChangeNeededInNsLabelSpec) {
			isChangeNeededInNsLabelSync = true
			if err := r.updateSyncNamespaceLabel(ctx, key, value, &namespaceLabel); err != nil {
				fmt.Println(err, "unable to update sync namespace label")
				return err
			}
		}
	}

	if isChangeNeededInNsLabelSpec {
		fmt.Println("stage 2.1:->")
		if err := r.Update(ctx, &namespaceLabel); err != nil {
			fmt.Println(err, "unable to update namespaceLabel")
			return err
		}
		return nil
	} else if isChangeNeededInNsLabelSync {
		fmt.Println("stage 2.2 or 2.3:->")
		if err := r.Status().Update(ctx, &namespaceLabel); err != nil {
			fmt.Println(err, "unable to update status of namespaceLabel")
			return err
		}
		return nil
	} else if !isChangeNeededInNamespace {
		//last step: update the namespace label's in order to the sync labels
		fmt.Println("Last step: ->")
		for key, value := range namespaceLabel.Status.SyncLabels {
			if err := r.updateNamespaceLabels(ctx, key, value, &namespace); err != nil {
				fmt.Println(err, "unable to update namespace labels")
				return err
			}
		}
		if err := r.Update(ctx, &namespace); err != nil {
			fmt.Println(err, "unable to update namespace")
			return err
		}
	}

	return nil
}

func (r *NamespaceLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	//get the nslabel
	var namespaceLabel omerv1.NamespaceLabel
	if err := r.Get(ctx, req.NamespacedName, &namespaceLabel); err != nil {
		fmt.Println("unable to fetch ns-label")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//check if  nslabel in deletion state
	if isNsLabelInDeletionState(namespaceLabel) {
		fmt.Println("the nslabel will deleted")
		//sent to clean all the labels from the namespace and then delete the nslabel finalizer
		r.cleanupNamespaceLabelFix(ctx, namespaceLabel, nsLabelFinalizer)
	} else {
		//first - add finalizer
		if !controllerutil.ContainsFinalizer(&namespaceLabel, nsLabelFinalizer) {
			controllerutil.AddFinalizer(&namespaceLabel, nsLabelFinalizer)
			if err := r.Update(ctx, &namespaceLabel); err != nil {
				return ctrl.Result{}, err
			}
		}
		//todo - sync nslabel to namespace object
		r.synchronizedNamespaceLabelToNamespaceFix(ctx, namespaceLabel)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&omerv1.NamespaceLabel{}).
		//Watches(
		//	&source.Kind{Type: &corev1.Namespace{}},
		//  handler.EnqueueRequestsFromMapFunc(r.)).
		Complete(r)
}
