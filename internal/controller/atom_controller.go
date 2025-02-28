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

	"github.com/go-logr/logr"

	v1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	yaml "sigs.k8s.io/yaml/goyaml.v3"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	atom_generator "github.com/pdok/atom-operator/internal/controller/atom_generator"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"

	traefikdynamic "github.com/traefik/traefik/v2/pkg/config/dynamic"
	traefikiov1alpha1 "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	valuesFileName  = "values.yaml"
	mainPortName    = "main"
	mainPortNr      = 80
	stripPrefixName = "atom-strip-prefix"
	headersName     = "atom-cors-headers"
	srvDir          = "/srv"
)

// AtomReconciler reconciles a Atom object
type AtomReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pdok.nl,resources=atoms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pdok.nl,resources=atoms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pdok.nl,resources=atoms/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=configmaps;services,verbs=watch;create;get;update;list;delete
// +kubebuilder:rbac:groups=traefik.io,resources=ingressroutes;middlewares,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=create;update;delete;list
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets/status,verbs=get;update
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets/finalizers,verbs=update

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

	result1, err1 := ReconcileEchoServer(r, ctx, req)
	if err1 != nil {
		return result1, err1
	}
	result2, err2 := ReconcileAtom(r, ctx, req)
	return result2, err2
}

func ReconcileEchoServer(r *AtomReconciler, ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
		if err := r.Create(ctx, deployment); err != nil {
			ll.Error(err, "failed to create Deployment")
			return ctrl.Result{}, err
		}
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
		if err := r.Create(ctx, service); err != nil {
			ll.Error(err, "failed to create Service")
			return ctrl.Result{}, err
		}
	}

	if err := r.Update(ctx, service); err != nil {
		ll.Error(err, "failed to update Service")
		return ctrl.Result{}, err
	}

	// Define the IngressRoute for Traefik
	ingressRoute := &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testecho-api",
			Namespace: testEcho.Namespace,
		},
		Spec: traefikiov1alpha1.IngressRouteSpec{
			Routes: []traefikiov1alpha1.Route{
				{
					Match: "Host(`localhost`) || Host(`kangaroo.test.pdok.nl`) && PathPrefix(`/testecho`)",
					Kind:  "Rule",
					Services: []traefikiov1alpha1.Service{
						{
							LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
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

		if err := r.Create(ctx, ingressRoute); err != nil {
			ll.Error(err, "failed to create IngressRoute")
			return ctrl.Result{}, err
		}
	}
	if err := r.Update(ctx, ingressRoute); err != nil {
		ll.Error(err, "failed to update IngressRoute")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func ReconcileAtom(r *AtomReconciler, ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ll := log.FromContext(ctx)

	// Fetch the Atom instance
	var atom pdoknlv3.Atom
	if err := r.Get(ctx, req.NamespacedName, &atom); err != nil {
		ll.Error(err, "unable to fetch Atom resource")
	}

	// Fetch the OwnerInfo instance
	var ownerInfo smoothoperatorv1.OwnerInfo

	if err := r.Get(ctx, client.ObjectKey{
		Namespace: atom.Namespace,
		Name:      atom.Spec.Service.OwnerInfoRef,
	}, &ownerInfo); err != nil {
		ll.Error(err, "unable to fetch OwnerInfo resource")
	}

	atomGeneratorConfig := GetGeneratorConfig(atom, ownerInfo, ll)
	if err := setupConfigMap(r, ctx, atom, ll, atomGeneratorConfig); err != nil {
		return ctrl.Result{}, err
	}

	if err := setupDeployment(r, ctx, atom, ll); err != nil {
		return ctrl.Result{}, err
	}

	if err := setupService(r, ctx, atom, ll); err != nil {
		return ctrl.Result{}, err
	}

	if err := setupPodDisruptionBudget(r, ctx, atom, ll); err != nil {
		return ctrl.Result{}, err
	}

	if err := setupIngressRoute(r, ctx, atom, ll); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil

}

func setupIngressRoute(r *AtomReconciler, ctx context.Context, atom pdoknlv3.Atom, ll logr.Logger) error {
	// TODO Middleware
	//  middlewareDownloads := &traefikiov1alpha1.Middleware{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      atom.Name + "-middleware",
	//		Namespace: atom.Namespace,
	//		Labels:    atom.Labels,
	//	},
	//	Spec: traefikiov1alpha1.MiddlewareSpec{
	//		ReplacePathRegex: &traefikdynamic.ReplacePathRegex{
	//			Regex: "^/{{ atom_uri }}/downloads/{{ item.version + '/' if item.version != '' else '' }}({{ download_links | json_query(blob_names_selecting_query) }})", // TODO
	//			Replacement: "/{{ item.blobPrefix }}/$1",
	//		},
	//	},
	// }

	middlewareHeaders := &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      atom.Name + headersName,
			Namespace: atom.Namespace,
			Labels:    atom.Labels,
		},
		Spec: traefikiov1alpha1.MiddlewareSpec{
			Headers: &traefikdynamic.Headers{
				AccessControlAllowHeaders:    []string{"Content-Type"},
				AccessControlAllowMethods:    []string{"GET", "HEAD", "OPTIONS"},
				AccessControlAllowOriginList: []string{"*"},
			},
		},
	}

	middlewareStripPrefix := &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name:      atom.Name + stripPrefixName,
			Namespace: atom.Namespace,
			Labels:    atom.Labels,
		},
		Spec: traefikiov1alpha1.MiddlewareSpec{
			StripPrefix: &traefikdynamic.StripPrefix{
				Prefixes: []string{atom.GetURI()},
			},
		},
	}

	middlewares := []*traefikiov1alpha1.Middleware{middlewareHeaders, middlewareStripPrefix}
	for _, middleware := range middlewares {
		// Set the controller reference for the Middleware
		if err := ctrl.SetControllerReference(&atom, middleware, r.Scheme); err != nil {
			ll.Error(err, "unable to set controller reference for Middleware")
			return err
		}

		key := types.NamespacedName{Namespace: middleware.GetNamespace(), Name: middleware.GetName()}
		if err := r.Get(ctx, key, middleware); err != nil {
			if client.IgnoreNotFound(err) != nil {
				ll.Error(err, "failed to get Middleware")
				return err
			}
			if err := r.Create(ctx, middleware); err != nil {
				ll.Error(err, "failed to create Middleware")
				return err
			}
		}
		if err := r.Update(ctx, middleware); err != nil {
			ll.Error(err, "failed to update Middleware")
			return err
		}
	}

	ingressRoute := &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      atom.Name + "-atom",
			Namespace: atom.Namespace,
			// Todo set uptime annotations
			//  Annotations: map[string]string{
			//	  "uptime.pdok.nl/id":   "{{ name|hash('sha1') }}",
			//	  "uptime.pdok.nl/name": "{{ uptime_name }}",
			//	  "uptime.pdok.nl/url":  "{{ ansible_env.BASE_URL }}/{{ atom_uri }}/index.xml",
			//	  "uptime.pdok.nl/tags": "public-stats,atom",
			//  },
			Labels: atom.Labels,
		},
		Spec: traefikiov1alpha1.IngressRouteSpec{
			Routes: []traefikiov1alpha1.Route{
				{
					Kind:  "Rule",
					Match: "Host(`localhost`) || Host(`kangaroo.test.pdok.nl`) && Path(`/" + atom.GetURI() + "/index.xml`)",
					Services: []traefikiov1alpha1.Service{
						{
							LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
								Name: atom.Name + "-atom-service",
								Port: intstr.IntOrString{Type: intstr.Int, IntVal: 80},
							},
						},
					},
					Middlewares: []traefikiov1alpha1.MiddlewareRef{
						{Name: atom.Name + headersName, Namespace: atom.Namespace},
						{Name: atom.Name + stripPrefixName, Namespace: atom.Namespace},
					},
				},
				// Todo loop per dataset
				// {
				//	Kind:  "Rule",
				//	Match: "Host(`localhost`) || Host(`kangaroo.test.pdok.nl`) && Path(`/" + atom.GetURI() + " /{{ dataset.name }}.xml`)",
				//	Services: []traefikiov1alpha1.Service{
				//		{
				//			LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
				//				Name: atom.Name + "-atom-service",
				//				Port: intstr.IntOrString{Type: intstr.Int, IntVal: 80},
				//			},
				//		},
				//	},
				//	Middlewares: []traefikiov1alpha1.MiddlewareRef{
				//		{Name: atom.Name + headersName, Namespace: atom.Namespace},
				//		{Name: atom.Name + stripPrefixName, Namespace: atom.Namespace},
				//	},
				// },
				{
					Kind:  "Rule",
					Match: "Host(`localhost`) || Host(`kangaroo.test.pdok.nl`) && PathPrefix(`/" + atom.GetURI() + "/downloads/`)",
					Services: []traefikiov1alpha1.Service{
						{LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
							Name:           "azure-storage",
							Port:           intstr.IntOrString{Type: intstr.String, StrVal: "azure-storage"},
							PassHostHeader: boolPtr(false),
						}},
					},
					Middlewares: []traefikiov1alpha1.MiddlewareRef{
						{Name: atom.Name + "-atom-headers", Namespace: atom.Namespace},
						// Todo loop per middleware download
					},
				},
			},
		},
	}

	// Set the controller reference for the ingress
	if err := ctrl.SetControllerReference(&atom, ingressRoute, r.Scheme); err != nil {
		ll.Error(err, "unable to set controller reference for IngressRoute")
		return err
	}

	// Create or update IngressRoute
	key := types.NamespacedName{Namespace: ingressRoute.GetNamespace(), Name: ingressRoute.GetName()}
	if err := r.Get(ctx, key, ingressRoute); err != nil {
		if client.IgnoreNotFound(err) != nil {
			ll.Error(err, "failed to get IngressRoute: %v")
			return err
		}

		if err := r.Create(ctx, ingressRoute); err != nil {
			ll.Error(err, "failed to create IngressRoute")
			return err
		}
	}
	if err := r.Update(ctx, ingressRoute); err != nil {
		ll.Error(err, "failed to update IngressRoute")
		return err
	}
	return nil
}

func setupPodDisruptionBudget(r *AtomReconciler, ctx context.Context, atom pdoknlv3.Atom, ll logr.Logger) error {
	podDisruptionBudget := &v1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      atom.Name + "-atom-pdb",
			Namespace: atom.Namespace,
			Labels:    atom.Labels},
		Spec: v1.PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		},
		Status: v1.PodDisruptionBudgetStatus{},
	}

	// Set the controller reference for the PodDisruptionBudget
	if err := ctrl.SetControllerReference(&atom, podDisruptionBudget, r.Scheme); err != nil {
		ll.Error(err, "unable to set controller reference for podDisruptionBudget")
		return err
	}

	key := types.NamespacedName{Namespace: podDisruptionBudget.Namespace, Name: podDisruptionBudget.Name}
	if err := r.Get(ctx, key, podDisruptionBudget); err != nil {
		if client.IgnoreNotFound(err) != nil {
			ll.Error(err, "failed to get PodDisruptionBudget")
			return err
		}
		if err := r.Create(ctx, podDisruptionBudget); err != nil {
			ll.Error(err, "failed to create PodDisruptionBudget")
			return err
		}
	}
	if err := r.Update(ctx, podDisruptionBudget); err != nil {
		ll.Error(err, "failed to update PodDisruptionBudget")
		return err
	}

	return nil
}

func setupService(r *AtomReconciler, ctx context.Context, atom pdoknlv3.Atom, ll logr.Logger) error {
	// Define the Service for the atom
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      atom.Name + "-atom-service",
			Namespace: atom.Namespace,
			Labels:    atom.Labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":           "atom-service",
				"service-type":  "atom",
				"dataset":       atom.Labels["dataset"],
				"dataset-owner": atom.Labels["dataset-owner"],
			},
			Ports: []corev1.ServicePort{{Name: "atom-service", Port: 80, TargetPort: intstr.FromInt32(80), Protocol: "TCP"}},
		},
	}

	// Set the controller reference for the Service
	if err := ctrl.SetControllerReference(&atom, service, r.Scheme); err != nil {
		ll.Error(err, "unable to set controller reference for service")
		return err
	}

	// Create or update Service
	key := types.NamespacedName{Namespace: service.GetNamespace(), Name: service.GetName()}
	if err := r.Get(ctx, key, service); err != nil {
		if client.IgnoreNotFound(err) != nil {
			ll.Error(err, "failed to get Service")
			return err
		}
		if err := r.Create(ctx, service); err != nil {
			ll.Error(err, "failed to create Service")
			return err
		}
	}

	if err := r.Update(ctx, service); err != nil {
		ll.Error(err, "failed to update Service")
		return err
	}
	return nil
}

func setupDeployment(r *AtomReconciler, ctx context.Context, atom pdoknlv3.Atom, ll logr.Logger) error {
	// Define the Deployment for the Atom
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      atom.Name + "-atom-service",
			Namespace: atom.Namespace,
			Labels:    atom.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 4},
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":           "atom-service",
					"service-type":  "atom",
					"dataset":       atom.Labels["dataset"],
					"dataset-owner": atom.Labels["dataset-owner"],
				},
			},
			RevisionHistoryLimit: int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"cluster-autoscaler.kubernetes.io/safe-to-evict": "true",
						"kubectl.kubernetes.io/default-container":        "atom-service",
						"priority.version-checker.io/atom-service":       "8",
						// Todo uptime.pdok.nl ?
					},
					Labels: map[string]string{
						"app":           "atom-service",
						"service-type":  "atom",
						"dataset":       atom.Labels["dataset"],
						"dataset-owner": atom.Labels["dataset-owner"]},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "socket", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: atom.Name + "-atom-generator",
							}}}},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "init-atom",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Image:           "acrpdokprodman.azurecr.io/mirror/docker.io/pdok/atom-generator:0.6.0",
							Command:         []string{"./atom"},
							Args:            []string{"-f=" + srvDir + "/config/" + valuesFileName, "-o=" + srvDir + "/data"},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: srvDir + "/data"},
								{Name: "config", MountPath: srvDir + "/config"},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "atom-service",
							Ports: []corev1.ContainerPort{
								{
									Name:          mainPortName,
									ContainerPort: mainPortNr,
								},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							Image:           "acrpdokprodman.azurecr.io/mirror/docker.io/pdok/lighttpd:1.4.67",
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/index.xml",
										Port:   intstr.FromInt32(mainPortNr),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/index.xml",
										Port:   intstr.FromInt32(mainPortNr),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("64M"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("0.01"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "socket", MountPath: "/tmp", ReadOnly: false},
								{Name: "data", MountPath: "var/www"},
							},
						},
					},
				},
			},
		},
	}

	// Set the controller reference to ensure garbage collection
	if err := ctrl.SetControllerReference(&atom, deployment, r.Scheme); err != nil {
		ll.Error(err, "unable to set controller reference for deployment")
		return err
	}

	// Create or update Deployment
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: deployment.GetNamespace(),
		Name:      deployment.GetName(),
	}, deployment); err != nil {
		if client.IgnoreNotFound(err) != nil {
			ll.Error(err, "failed to get Deployment")
			return err
		}
		if err := r.Create(ctx, deployment); err != nil {
			ll.Error(err, "failed to create Deployment")
			return err
		}
	}
	if err := r.Update(ctx, deployment); err != nil {
		return err
	}
	return nil
}

func setupConfigMap(r *AtomReconciler, ctx context.Context, atom pdoknlv3.Atom, ll logr.Logger, generatorConfig string) error {

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      atom.Name + "-atom-generator",
			Namespace: atom.Namespace,
		},
		Immutable: boolPtr(true),
		Data:      map[string]string{valuesFileName: generatorConfig},
	}

	// Create or update ConfigMap
	key := types.NamespacedName{Namespace: configMap.GetNamespace(), Name: configMap.GetName()}
	if err := r.Get(ctx, key, configMap); err != nil {
		if client.IgnoreNotFound(err) != nil {
			ll.Error(err, "failed to get ConfigMap")
			return err
		}
		if err := r.Create(ctx, configMap); err != nil {
			ll.Error(err, "failed to create ConfigMap")
			return err
		}
	}
	err := r.Update(ctx, configMap)
	if err != nil {
		return err
	}

	// Set the controller reference for the ConfigMap
	if err := ctrl.SetControllerReference(&atom, configMap, r.Scheme); err != nil {
		ll.Error(err, "unable to set ConfigMap reference for service")
		return err
	}
	return nil
}

func GetGeneratorConfig(atom pdoknlv3.Atom, ownerInfo smoothoperatorv1.OwnerInfo, ll logr.Logger) string {

	atomGeneratorConfig, err := atom_generator.MapAtomV3ToAtomGeneratorConfig(atom, ownerInfo)
	if err != nil {
		ll.Error(err, "failed to map the V3 atom to generator config.")
	}

	yamlConfig, err := yaml.Marshal(&atomGeneratorConfig)
	if err != nil {
		ll.Error(err, "failed to marshal the V3 atom generator config to yaml")
	}
	return string(yamlConfig)
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
		Owns(&v1.PodDisruptionBudget{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&smoothoperatorv1.OwnerInfo{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
