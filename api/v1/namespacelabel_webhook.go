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

package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// log is for logging in this package.
var namespacelabellog = logf.Log.WithName("namespacelabel-resource")

func (r *NamespaceLabel) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-omer-omer-io-v1-namespacelabel,mutating=true,failurePolicy=fail,sideEffects=None,groups=omer.omer.io,resources=namespacelabels,verbs=create;update,versions=v1,name=mnamespacelabel.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &NamespaceLabel{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NamespaceLabel) Default() {
	namespacelabellog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-omer-omer-io-v1-namespacelabel,mutating=false,failurePolicy=fail,sideEffects=None,groups=omer.omer.io,resources=namespacelabels,verbs=create;update,versions=v1,name=vnamespacelabel.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NamespaceLabel{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NamespaceLabel) ValidateCreate() error {
	namespacelabellog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.validateNamespaceLabel()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NamespaceLabel) ValidateUpdate(old runtime.Object) error {
	namespacelabellog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NamespaceLabel) ValidateDelete() error {
	namespacelabellog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *NamespaceLabel) validateNamespaceLabel() error {
	var allErrs field.ErrorList
	if err := r.validateNamespaceLabelName(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "omer.io", Kind: "NamespaceLabel"},
		r.Name, allErrs)
}

func (r *NamespaceLabel) validateNamespaceLabelName() *field.Error {
	if r.ObjectMeta.Name != r.ObjectMeta.Namespace {
		return field.Invalid(field.NewPath("metadata").Child("name"), r.Name, "the name of the namespacelabel needs to be like the name of the namespace")
	}
	return nil
}
