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
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	omerv1 "omer.io/namespacelabel/api/v1"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"

	"golang.org/x/exp/slices"
)

// NamespaceLabelReconciler reconciles a NamespaceLabel object
type NamespaceLabelReconciler struct {
	client.Client
	Logger          logr.Logger
	Scheme          *runtime.Scheme
	ProtectedLabels []string
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

// the function return all the keys that exist in map1 and not in map2
func getDiffBetweenMaps(map1 map[string]string, map2 map[string]string) map[string]string {
	diffMap := make(map[string]string)
	for key, value := range map1 {
		if !isLabelKeyExistInLabels(map2, key) {
			diffMap[key] = value
		}
	}

	return diffMap
}

func (r *NamespaceLabelReconciler) cleanupNamespaceLabel(ctx context.Context, namespaceLabel omerv1.NamespaceLabel, nsLabelFinalizer string) error {
	//get the namespace for sync to nslabel
	var namespace v1.Namespace
	namespacedName := types.NamespacedName{Name: namespaceLabel.Namespace}
	if err := r.Get(ctx, namespacedName, &namespace); err != nil {
		r.Logger.Error(err, "unable to fetch namespace", namespaceLabel.Namespace)
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
			r.Logger.Error(err, "unable to update namespace", namespace.Name)
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

// the main function for handling the sync between the cr and the namespace
func (r *NamespaceLabelReconciler) handleSyncNamespaceLabel(ctx context.Context, namespaceLabel omerv1.NamespaceLabel) error {
	//get the namespace for sync to nslabel
	var namespace v1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: namespaceLabel.Namespace}, &namespace); err != nil {
		r.Logger.Error(err, "unable to fetch namespace", namespace.ObjectMeta.Name)
		return client.IgnoreNotFound(err)
	}

	postSyncLabels, postUnSyncLabels := r.sortSyncLabels(namespaceLabel, namespace)
	fmt.Println(postSyncLabels, postUnSyncLabels)
	if err := r.syncNamespaceToNamespaceLabel(ctx, namespaceLabel, namespace, postSyncLabels); err != nil {
		r.Logger.Error(err, "unable to update namespacelabel in order to the namespacelabel", namespaceLabel.ObjectMeta.Name)
		return err
	}

	namespaceLabel.Status.SyncLabels = postSyncLabels
	namespaceLabel.Status.UnSyncLabels = postUnSyncLabels
	if err := r.Status().Update(ctx, &namespaceLabel); err != nil {
		r.Logger.Error(err, "unable to update status of namespaceLabel", namespaceLabel.ObjectMeta.Name)
		return err
	}

	return nil
}

func (r *NamespaceLabelReconciler) sortSyncLabels(namespaceLabel omerv1.NamespaceLabel, namespace v1.Namespace) (syncLabels map[string]string, unSyncLabels map[string]string) {
	syncLabels = make(map[string]string)
	unSyncLabels = make(map[string]string)

	//stage 2:running on all the labels in the spec of the nslabel object
	//stage 2.1: the label is in the namespace and not in the sync - result: update the Unsync
	//stage 2.2: the label is in the namespace and in the sync - result: update the sync
	//stage 2.3: the label is in the not in the namespace and also not in the sync - result: update the sync

	for key, value := range namespaceLabel.Spec.Labels {
		if slices.Contains(r.ProtectedLabels, key) {
			unSyncLabels[key] = value
		} else {
			if isLabelKeyExistInLabels(namespace.ObjectMeta.Labels, key) {
				if isLabelKeyExistInLabels(namespaceLabel.Status.SyncLabels, key) {
					syncLabels[key] = value
				} else {
					unSyncLabels[key] = value
				}
			} else {
				syncLabels[key] = value
			}
		}

	}
	return syncLabels, unSyncLabels
}

func (r *NamespaceLabelReconciler) syncNamespaceToNamespaceLabel(ctx context.Context, namespaceLabel omerv1.NamespaceLabel, namespace v1.Namespace, postSyncLabels map[string]string) error {
	newNamespaceLabels := make(map[string]string)
	deletedLabels := getDiffBetweenMaps(namespaceLabel.Status.SyncLabels, postSyncLabels)

	for key, value := range namespace.ObjectMeta.Labels {
		if isLabelKeyExistInLabels(deletedLabels, key) {
			continue
		} else {
			newNamespaceLabels[key] = value
		}
	}

	for key, value := range postSyncLabels {
		newNamespaceLabels[key] = value
	}

	namespace.SetLabels(newNamespaceLabels)
	if err := r.Update(ctx, &namespace); err != nil {
		r.Logger.Error(err, "unable to update namespace labels", namespace.Name)
		return err
	}

	return nil
}

func (r *NamespaceLabelReconciler) listAllNamespaceLabel(namespace client.Object) []reconcile.Request {
	namespaceLabelList := &omerv1.NamespaceLabelList{}
	listOps := &client.ListOptions{
		Namespace: namespace.GetNamespace(),
	}
	err := r.List(context.TODO(), namespaceLabelList, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(namespaceLabelList.Items))
	for i, item := range namespaceLabelList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

func (r *NamespaceLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger = ctrllog.FromContext(ctx)

	// TODO(user): your logic here

	//get the nslabel
	var namespaceLabel omerv1.NamespaceLabel
	if err := r.Get(ctx, req.NamespacedName, &namespaceLabel); err != nil {
		r.Logger.Error(err, "unable to fetch ns-label", "namespace", namespaceLabel.ObjectMeta.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//check if  nslabel in deletion state
	if isNsLabelInDeletionState(namespaceLabel) {
		r.Logger.Info("NamespaceLabel in deletion state ")
		//sent to clean all the labels from the namespace and then delete the nslabel finalizer
		r.cleanupNamespaceLabel(ctx, namespaceLabel, nsLabelFinalizer)
	} else {
		//first - add finalizer
		if !controllerutil.ContainsFinalizer(&namespaceLabel, nsLabelFinalizer) {
			controllerutil.AddFinalizer(&namespaceLabel, nsLabelFinalizer)
			if err := r.Update(ctx, &namespaceLabel); err != nil {
				return ctrl.Result{}, err
			}
		}
		//todo - sync nslabel to namespace object
		r.handleSyncNamespaceLabel(ctx, namespaceLabel)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&omerv1.NamespaceLabel{}).
		Watches(
			&source.Kind{Type: &v1.Namespace{}},
			handler.EnqueueRequestsFromMapFunc(r.listAllNamespaceLabel),
		).
		Complete(r)
}
