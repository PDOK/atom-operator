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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	traefikv1 "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AtomReconciler reconciles a Atom object
type AtomReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pdok.nl,resources=atoms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pdok.nl,resources=atoms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pdok.nl,resources=atoms/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=create;get;update;list;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=create;get;update;list;delete
// +kubebuilder:rbac:groups=traefik.containo.us,resources=ingressroutes,verbs=create;get;update;list;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Atom object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *AtomReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ll := log.FromContext(ctx)

	// Fetch the Atom instance
	var testEcho pdoknlv3.Atom
	if err := r.Get(ctx, req.NamespacedName, &testEcho); err != nil {
		ll.Error(err, "unable to fetch TestEcho")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Define the Deployment for the echo server
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testecho-server",
			Namespace: testEcho.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "testecho"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "testecho"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "echo-server",
							Image: "ealen/echo-server",
							Ports: []corev1.ContainerPort{{ContainerPort: 80}},
						},
					},
				},
			},
		},
	}

	// Set the controller reference to ensure garbage collection
	if err := ctrl.SetControllerReference(&testEcho, deployment, r.Scheme); err != nil {
		ll.Error(err, "unable to set controller reference for deployment")
		return ctrl.Result{}, err
	}

	// Create or update Deployment
	key := types.NamespacedName{Namespace: deployment.GetNamespace(), Name: deployment.GetName()}
	if err := r.Get(ctx, key, deployment); err != nil {
		if client.IgnoreNotFound(err) != nil {
			ll.Error(err, "failed to get object")
			return ctrl.Result{}, err
		}
		r.Create(ctx, deployment)
	}
	err := r.Update(ctx, deployment)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Define the Service for the echo server
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testecho-api",
			Namespace: testEcho.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "testecho"},
			Ports:    []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt32(80)}},
		},
	}

	// Set the controller reference for the service
	if err := ctrl.SetControllerReference(&testEcho, service, r.Scheme); err != nil {
		ll.Error(err, "unable to set controller reference for service")
		return ctrl.Result{}, err
	}

	// Create or update Service
	key = types.NamespacedName{Namespace: service.GetNamespace(), Name: service.GetName()}
	if err := r.Get(ctx, key, service); err != nil {
		if client.IgnoreNotFound(err) != nil {
			ll.Error(err, "failed to get object")
			return ctrl.Result{}, err
		}
		r.Create(ctx, service)
	}
	r.Update(ctx, service)

	// Define the IngressRoute for Traefik
	ingressRoute := &traefikv1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testecho-api",
			Namespace: testEcho.Namespace,
		},
		Spec: traefikv1.IngressRouteSpec{
			Routes: []traefikv1.Route{
				{
					Match: "Host(`localhost`) || Host(`kangaroo.test.pdok.nl`) && PathPrefix(`/testecho`)",
					Kind:  "Rule",
					Services: []traefikv1.Service{
						{
							LoadBalancerSpec: traefikv1.LoadBalancerSpec{
								Name: "testecho-api",
								Port: intstr.FromInt32(80),
							},
						},
					},
				},
			},
		},
	}

	// Set the controller reference for the ingress
	if err := ctrl.SetControllerReference(&testEcho, ingressRoute, r.Scheme); err != nil {
		ll.Error(err, "unable to set controller reference for ingress")
		return ctrl.Result{}, err
	}

	// Create or update IngressRoute

	key = types.NamespacedName{Namespace: ingressRoute.GetNamespace(), Name: ingressRoute.GetName()}
	if err := r.Get(ctx, key, ingressRoute); err != nil {
		if client.IgnoreNotFound(err) != nil {
			ll.Error(err, "failed to get object: %v")
			return ctrl.Result{}, err
		}
		r.Create(ctx, ingressRoute)
	}
	r.Update(ctx, ingressRoute)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AtomReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pdoknlv3.Atom{}).
		Named("atom").
		Complete(r)
}
