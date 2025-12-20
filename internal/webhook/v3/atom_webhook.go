/*
Copyright 2025.

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

package v3

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

// nolint:unused
// log is for logging in this package.
var atomlog = logf.Log.WithName("atom-resource")

// SetupAtomWebhookWithManager registers the webhook for Atom in the manager.
func SetupAtomWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&pdoknlv3.Atom{}).
		WithValidator(&AtomCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-pdok-nl-v3-atom,mutating=false,failurePolicy=fail,sideEffects=None,groups=pdok.nl,resources=atoms,verbs=create;update,versions=v3,name=vatom-v3.kb.io,admissionReviewVersions=v1

// AtomCustomValidator struct is responsible for validating the Atom resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AtomCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &AtomCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Atom.
func (v *AtomCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	atom, ok := obj.(*pdoknlv3.Atom)
	if !ok {
		return nil, fmt.Errorf("expected a Atom object but got %T", obj)
	}
	atomlog.Info("Validation for Atom upon creation", "name", atom.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Atom.
func (v *AtomCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	atom, ok := newObj.(*pdoknlv3.Atom)
	if !ok {
		return nil, fmt.Errorf("expected a Atom object for the newObj but got %T", newObj)
	}
	atomlog.Info("Validation for Atom upon update", "name", atom.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Atom.
func (v *AtomCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	atom, ok := obj.(*pdoknlv3.Atom)
	if !ok {
		return nil, fmt.Errorf("expected a Atom object but got %T", obj)
	}
	atomlog.Info("Validation for Atom upon deletion", "name", atom.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
