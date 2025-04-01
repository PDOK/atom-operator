/*
MIT License

Copyright (c) 2024 Publieke Dienstverlening op de Kaart

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package v3

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

// log is for logging in this package.
//

var atomlog = logf.Log.WithName("atom-resource")

// SetupAtomWebhookWithManager registers the webhook for Atom in the manager.
func SetupAtomWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&pdoknlv3.Atom{}).
		WithValidator(&AtomCustomValidator{mgr.GetClient()}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-pdok-nl-v3-atom,mutating=false,failurePolicy=fail,sideEffects=None,groups=pdok.nl,resources=atoms,verbs=create;update,versions=v3,name=vatom-v3.kb.io,admissionReviewVersions=v1

// AtomCustomValidator struct is responsible for validating the Atom resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AtomCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &AtomCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Atom.
func (v *AtomCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	atom, ok := obj.(*pdoknlv3.Atom)
	if !ok {
		return nil, fmt.Errorf("expected a Atom object but got %T", obj)
	}
	atomlog.Info("Validation for Atom upon creation", "name", atom.GetName())

	return atom.ValidateCreate(v.Client)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Atom.
func (v *AtomCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	atomlog.Info("reading newAtom")
	atom, ok := newObj.(*pdoknlv3.Atom)
	if !ok {
		return nil, fmt.Errorf("expected a Atom object for the newObj but got %T", newObj)
	}
	atomlog.Info("reading oldAtom")
	atomOld, ok := oldObj.(*pdoknlv3.Atom)
	if !ok {
		return nil, fmt.Errorf("expected a Atom object for the oldObj but got %T", oldObj)
	}
	atomlog.Info("Validation for Atom upon update", "name", atom.GetName())

	return atom.ValidateUpdate(v.Client, atomOld)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Atom.
func (v *AtomCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	atom, ok := obj.(*pdoknlv3.Atom)
	if !ok {
		return nil, fmt.Errorf("expected a Atom object but got %T", obj)
	}
	atomlog.Info("Validation for Atom upon deletion", "name", atom.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
