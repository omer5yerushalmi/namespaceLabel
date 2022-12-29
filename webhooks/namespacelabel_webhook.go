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

package webhooks

import (
	"context"

	"github.com/go-logr/logr"
	"net/http"
	omerv1 "omer.io/namespacelabel/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type NamespaceLabelWebhook struct {
	Decoder *admission.Decoder
	Log     logr.Logger
}

const denyMessageValidateNamespaceLabel = "the name of the namespacelabel needs to be like the name of the namespace - low"
const allowMessageValidateNamespaceLabel = "good job"

// +kubebuilder:webhook:path=/validate-omer-omer-io-v1-namespacelabel,mutating=false,failurePolicy=fail,sideEffects=None,groups=omer.omer.io,resources=namespacelabels,verbs=create;update,versions=v1,name=vnamespacelabel.kb.io,admissionReviewVersions=v1
func (a *NamespaceLabelWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := a.Log.WithValues("webhook", "Namespacelabel Webhook", "Name", req.Name)
	log.Info("webhook request received")

	// Get the incoming Namespacelabel.
	namespaceLabel := &omerv1.NamespaceLabel{}
	err := a.Decoder.Decode(req, namespaceLabel)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if namespaceLabel.ObjectMeta.Name != namespaceLabel.ObjectMeta.Namespace {
		return admission.Denied(denyMessageValidateNamespaceLabel)
	}
	return admission.Allowed(allowMessageValidateNamespaceLabel)
}
