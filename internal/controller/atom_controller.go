/*
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

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	policyv1 "k8s.io/api/policy/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"

	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	reconciledConditionType         = "Reconciled"
	reconciledConditionReasonSucces = "Succes"
	reconciledConditionReasonError  = "Error"
)

const (
	appLabelKey       = "app"
	appName           = "atom-service"
	configFileName    = "values.yaml"
	atomPortName      = "atom-service"
	atomPortNr        = 80
	stripPrefixSuffix = "-atom-prefixstrip"
	headersSuffix     = "-atom-headers"
	downloadsSuffix   = "-atom-downloads-"
	nameSuffix        = "-atom"
	generatorSuffix   = "-atom-generator"

	srvDir = "/srv"
)

// AtomReconciler reconciles a Atom object
type AtomReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	AtomGeneratorImage string
	LighttpdImage      string
	CSP                string
}

// +kubebuilder:rbac:groups=pdok.nl,resources=atoms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pdok.nl,resources=atoms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pdok.nl,resources=atoms/finalizers,verbs=update
// +kubebuilder:rbac:groups=pdok.nl,resources=ownerinfo,verbs=get;list;watch
// +kubebuilder:rbac:groups=pdok.nl,resources=ownerinfo/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=configmaps;services,verbs=watch;create;get;update;list;delete
// +kubebuilder:rbac:groups=traefik.io,resources=ingressroutes;middlewares,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=create;update;delete;list;watch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets/status,verbs=get;update
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// The Reconcile function compares the state specified by
// the Atom object against the actual cluster state, and then
// performs operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *AtomReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	lgr := logf.FromContext(ctx)
	lgr.Info("Starting reconcile for atom resource", "name", req.NamespacedName)

	lgr.Info("Fetching atom", "name", req.NamespacedName)
	// Fetch the Atom instance
	atom := &pdoknlv3.Atom{}
	if err = r.Client.Get(ctx, req.NamespacedName, atom); err != nil {
		if apierrors.IsNotFound(err) {
			lgr.Info("Atom resource not found", "name", req.NamespacedName)
		} else {
			lgr.Error(err, "unable to fetch Atom resource", "error", err)
		}
		return result, client.IgnoreNotFound(err)
	}

	lgr.Info("Fetching OwnerInfo", "name", req.NamespacedName)
	// Fetch the OwnerInfo instance
	ownerInfo := &smoothoperatorv1.OwnerInfo{}
	objectKey := client.ObjectKey{
		Namespace: atom.Namespace,
		Name:      atom.Spec.Service.OwnerInfoRef,
	}
	if err = r.Client.Get(ctx, objectKey, ownerInfo); err != nil {
		if apierrors.IsNotFound(err) {
			lgr.Info("OwnerInfo resource not found", "name", req.NamespacedName)
		} else {
			lgr.Error(err, "unable to fetch OwnerInfo resource", "error", err)
		}
		return result, client.IgnoreNotFound(err)
	}

	// Recover from a panic so we can add the error to the status of the Atom
	defer func() {
		if rec := recover(); rec != nil {
			err = recoveredPanicToError(rec)
			r.logAndUpdateStatusError(ctx, atom, err)
		}
	}()

	// Check TTL expiry
	if ttlExpired(atom) {
		err = r.Client.Delete(ctx, atom)

		return result, err
	}

	lgr.Info("creating resources for atom", "atom", atom)
	operationResults, err := r.createOrUpdateAllForAtom(ctx, atom, ownerInfo)
	if err != nil {
		lgr.Info("failed creating resources for atom", "atom", atom)
		r.logAndUpdateStatusError(ctx, atom, err)
		return result, err
	}
	lgr.Info("finished creating resources for atom", "atom", atom)
	r.logAndUpdateStatusFinished(ctx, atom, operationResults)

	return result, err
}

func (r *AtomReconciler) createOrUpdateAllForAtom(ctx context.Context, atom *pdoknlv3.Atom, ownerInfo *smoothoperatorv1.OwnerInfo) (operationResults map[string]controllerutil.OperationResult, err error) {
	operationResults = make(map[string]controllerutil.OperationResult)
	c := r.Client

	// region Create or update ConfigMap
	configMap := getBareConfigMap(atom)

	// mutate (also) before to get the hash suffix in the name
	if err = r.mutateAtomGeneratorConfigMap(atom, ownerInfo, configMap); err != nil {
		return operationResults, err
	}
	operationResults[smoothutil.GetObjectFullName(r.Client, atom)], err = controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		return r.mutateAtomGeneratorConfigMap(atom, ownerInfo, configMap)
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothutil.GetObjectFullName(c, configMap), err)
	}
	// endregion

	// region Create or update Deployment
	deployment := getBareDeployment(atom)
	operationResults[smoothutil.GetObjectFullName(r.Client, deployment)], err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		return r.mutateDeployment(atom, deployment, configMap.GetName())
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothutil.GetObjectFullName(c, deployment), err)
	}
	// endregion

	// region Create or update Service
	service := getBareService(atom)
	operationResults[smoothutil.GetObjectFullName(r.Client, service)], err = controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		return r.mutateService(atom, service)
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothutil.GetObjectFullName(c, service), err)
	}
	// endregion

	// region Create or update Middleware

	stripPrefixMiddleware := getBareStripPrefixMiddleware(atom)
	operationResults[smoothutil.GetObjectFullName(r.Client, stripPrefixMiddleware)], err = controllerutil.CreateOrUpdate(ctx, r.Client, stripPrefixMiddleware, func() error {
		return r.mutateStripPrefixMiddleware(atom, stripPrefixMiddleware)
	})
	if err != nil {
		return operationResults, fmt.Errorf("could not create or update resource %s: %w", smoothutil.GetObjectFullName(c, stripPrefixMiddleware), err)
	}

	corsHeadersMiddleware := getBareHeadersMiddleware(atom)
	operationResults[smoothutil.GetObjectFullName(r.Client, corsHeadersMiddleware)], err = controllerutil.CreateOrUpdate(ctx, r.Client, corsHeadersMiddleware, func() error {
		return r.mutateHeadersMiddleware(atom, corsHeadersMiddleware, r.CSP)
	})
	if err != nil {
		return operationResults, fmt.Errorf("could not create or update resource %s: %w", smoothutil.GetObjectFullName(c, corsHeadersMiddleware), err)
	}

	// Create or update extra middleware per downloadLink
	for prefix, group := range getDownloadLinkGroups(atom.GetDownloadLinks()) {
		downloadLinkMiddleware := getBareDownloadLinkMiddleware(atom, *group.index)
		operationResults[smoothutil.GetObjectFullName(r.Client, downloadLinkMiddleware)], err = controllerutil.CreateOrUpdate(ctx, r.Client, downloadLinkMiddleware, func() error {
			return r.mutateDownloadLinkMiddleware(atom, prefix, group.files, downloadLinkMiddleware)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothutil.GetObjectFullName(c, downloadLinkMiddleware), err)
		}
	}

	// endregion

	// region Create or update IngressRoute
	ingressRoute := getBareIngressRoute(atom)
	operationResults[smoothutil.GetObjectFullName(r.Client, ingressRoute)], err = controllerutil.CreateOrUpdate(ctx, r.Client, ingressRoute, func() error {
		return r.mutateIngressRoute(atom, ingressRoute)
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothutil.GetObjectFullName(c, ingressRoute), err)
	}

	// endregion

	// region Create or update PodDisruptionBudget
	podDisruptionBudget := getBarePodDisruptionBudget(atom)
	operationResults[smoothutil.GetObjectFullName(r.Client, podDisruptionBudget)], err = controllerutil.CreateOrUpdate(ctx, r.Client, podDisruptionBudget, func() error {
		return r.mutatePodDisruptionBudget(atom, podDisruptionBudget)
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothutil.GetObjectFullName(c, podDisruptionBudget), err)
	}
	// endregion

	return operationResults, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AtomReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pdoknlv3.Atom{}).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&traefikiov1alpha1.Middleware{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&traefikiov1alpha1.IngressRoute{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&policyv1.PodDisruptionBudget{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&smoothoperatorv1.OwnerInfo{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

func getLabels(atom *pdoknlv3.Atom) map[string]string {
	labels := smoothutil.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = appName
	return labels
}

func ttlExpired(atom *pdoknlv3.Atom) bool {
	if lifecycle := atom.Spec.Lifecycle; lifecycle != nil && lifecycle.TTLInDays != nil {
		expiresAt := atom.GetCreationTimestamp().Add(time.Duration(*lifecycle.TTLInDays) * 24 * time.Hour)

		return expiresAt.Before(time.Now())
	}

	return false
}

func recoveredPanicToError(rec any) (err error) {
	switch x := rec.(type) {
	case string:
		err = errors.New(x)
	case error:
		err = x
	default:
		err = errors.New("unknown panic")
	}

	// Add stack
	err = errors.WithStack(err)

	return
}
