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
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	yaml "sigs.k8s.io/yaml/goyaml.v3"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	"github.com/pdok/atom-operator/internal/controller/generator"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"

	traefikdynamic "github.com/traefik/traefik/v3/pkg/config/dynamic"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	reconciledConditionType         = "Reconciled"
	reconciledConditionReasonSucces = "Succes"
	reconciledConditionReasonError  = "Error"
)

const (
	controllerName  = "atom-controller"
	appLabelKey     = "app"
	atomName        = "atom-service"
	configFileName  = "values.yaml"
	atomPortName    = "atom-service"
	atomPortNr      = 80
	stripPrefixName = "atom-strip-prefix"
	corsHeadersName = "atom-cors-headers"
	downloadsName   = "atom-downloads"

	srvDir = "/srv"
)

var (
	finalizerName = controllerName + "." + pdoknlv3.GroupVersion.Group + "/finalizer"
)

// AtomReconciler reconciles a Atom object
type AtomReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	AtomGeneratorImage string
	LighttpdImage      string
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
	lgr := log.FromContext(ctx)
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
	if err := r.Client.Get(ctx, objectKey, ownerInfo); err != nil {
		if apierrors.IsNotFound(err) {
			lgr.Info("OwnerInfo resource not found", "name", req.NamespacedName)
		} else {
			lgr.Error(err, "unable to fetch OwnerInfo resource", "error", err)
		}
		return result, client.IgnoreNotFound(err)
	}

	lgr.Info("Get object full name")
	fullName := smoothoperatorutils.GetObjectFullName(r.Client, atom)
	lgr.Info("Finalize if necessary")
	shouldContinue, err := smoothoperatorutils.FinalizeIfNecessary(ctx, r.Client, atom, finalizerName, func() error {
		lgr.Info("deleting resources", "name", fullName)
		return r.deleteAllForAtom(ctx, atom, ownerInfo)
	})
	if !shouldContinue || err != nil {
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

func (r *AtomReconciler) logAndUpdateStatusError(ctx context.Context, atom *pdoknlv3.Atom, err error) {
	r.updateStatus(ctx, atom, []metav1.Condition{{
		Type:               reconciledConditionType,
		Status:             metav1.ConditionFalse,
		Reason:             reconciledConditionReasonError,
		Message:            err.Error(),
		ObservedGeneration: atom.Generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}}, nil)
}

func (r *AtomReconciler) createOrUpdateAllForAtom(ctx context.Context, atom *pdoknlv3.Atom, ownerInfo *smoothoperatorv1.OwnerInfo) (operationResults map[string]controllerutil.OperationResult, err error) {
	operationResults = make(map[string]controllerutil.OperationResult)
	c := r.Client

	// region Create or update ConfigMap
	configMap := getBareConfigMap(atom)

	// mutate (also) before to get the hash suffix in the name
	if err = r.mutateConfigMap(atom, ownerInfo, configMap); err != nil {
		return operationResults, err
	}
	operationResults[smoothoperatorutils.GetObjectFullName(r.Client, atom)], err = controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		return r.mutateConfigMap(atom, ownerInfo, configMap)
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, configMap), err)
	}
	// endregion

	// region Create or update Deployment
	deployment := getBareDeployment(atom)
	operationResults[smoothoperatorutils.GetObjectFullName(r.Client, deployment)], err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		return r.mutateDeployment(atom, deployment, configMap.GetName())
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, deployment), err)
	}
	// endregion

	// region Create or update Service
	service := getBareService(atom)
	operationResults[smoothoperatorutils.GetObjectFullName(r.Client, service)], err = controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		return r.mutateService(atom, service)
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, service), err)
	}
	// endregion

	// region Create or update Middleware

	stripPrefixMiddleware := getBareStripPrefixMiddleware(atom)
	operationResults[smoothoperatorutils.GetObjectFullName(r.Client, stripPrefixMiddleware)], err = controllerutil.CreateOrUpdate(ctx, r.Client, stripPrefixMiddleware, func() error {
		return r.mutateStripPrefixMiddleware(atom, stripPrefixMiddleware)
	})
	if err != nil {
		return operationResults, fmt.Errorf("could not create or update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, stripPrefixMiddleware), err)
	}

	corsHeadersMiddleware := getBareCorsHeadersMiddleware(atom)
	operationResults[smoothoperatorutils.GetObjectFullName(r.Client, corsHeadersMiddleware)], err = controllerutil.CreateOrUpdate(ctx, r.Client, corsHeadersMiddleware, func() error {
		return r.mutateCorsHeadersMiddleware(atom, corsHeadersMiddleware)
	})
	if err != nil {
		return operationResults, fmt.Errorf("could not create or update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, corsHeadersMiddleware), err)
	}

	// Create or update extra middleware per downloadLink
	for index, downloadLink := range atom.GetIndexedDownloadLinks() {
		downloadLinkMiddleware := getBareDownloadLinkMiddleware(atom, index)
		operationResults[smoothoperatorutils.GetObjectFullName(r.Client, downloadLinkMiddleware)], err = controllerutil.CreateOrUpdate(ctx, r.Client, downloadLinkMiddleware, func() error {
			return r.mutateDownloadLinkMiddleware(atom, &downloadLink, downloadLinkMiddleware)
		})
		if err != nil {
			return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, downloadLinkMiddleware), err)
		}
	}

	// endregion

	// region Create or update IngressRoute
	ingressRoute := getBareIngressRoute(atom)
	operationResults[smoothoperatorutils.GetObjectFullName(r.Client, ingressRoute)], err = controllerutil.CreateOrUpdate(ctx, r.Client, ingressRoute, func() error {
		return r.mutateIngressRoute(atom, ingressRoute)
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, ingressRoute), err)
	}

	// endregion

	// region Create or update PodDisruptionBudget
	podDisruptionBudget := getBarePodDisruptionBudget(atom)
	operationResults[smoothoperatorutils.GetObjectFullName(r.Client, podDisruptionBudget)], err = controllerutil.CreateOrUpdate(ctx, r.Client, podDisruptionBudget, func() error {
		return r.mutatePodDisruptionBudget(atom, podDisruptionBudget)
	})
	if err != nil {
		return operationResults, fmt.Errorf("unable to create/update resource %s: %w", smoothoperatorutils.GetObjectFullName(c, podDisruptionBudget), err)
	}
	// endregion

	return operationResults, nil
}

func (r *AtomReconciler) deleteAllForAtom(ctx context.Context, atom *pdoknlv3.Atom, ownerInfo *smoothoperatorv1.OwnerInfo) (err error) {
	configMap := getBareConfigMap(atom)
	// mutate (also) before to get the hash suffix in the name
	if err = r.mutateConfigMap(atom, ownerInfo, configMap); err != nil {
		return
	}
	objects := []client.Object{
		configMap,
		getBareDeployment(atom),
		getBareService(atom),
		getBareStripPrefixMiddleware(atom),
		getBareCorsHeadersMiddleware(atom),
		getBareIngressRoute(atom),
		getBarePodDisruptionBudget(atom),
	}
	for index := range atom.GetIndexedDownloadLinks() {
		objects = append(objects, getBareDownloadLinkMiddleware(atom, index))
	}

	return smoothoperatorutils.DeleteObjects(ctx, r.Client, objects)
}

func getBareConfigMap(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getBareDeployment(obj).GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateConfigMap(atom *pdoknlv3.Atom, ownerInfo *smoothoperatorv1.OwnerInfo, configMap *corev1.ConfigMap) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = atomName
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		generatorConfig, err := getGeneratorConfig(atom, ownerInfo)
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{configFileName: generatorConfig}
	}
	configMap.Immutable = smoothoperatorutils.Pointer(true)

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(atom, configMap, r.Scheme); err != nil {
		return err
	}
	return smoothoperatorutils.AddHashSuffix(configMap)
}

func getBareDeployment(obj metav1.Object) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + atomName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

//nolint:funlen
func (r *AtomReconciler) mutateDeployment(atom *pdoknlv3.Atom, deployment *appsv1.Deployment, configMapName string) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = atomName
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, deployment, labels); err != nil {
		return err
	}

	podTemplateAnnotations := smoothoperatorutils.CloneOrEmptyMap(deployment.Spec.Template.GetAnnotations())

	matchLabels := smoothoperatorutils.CloneOrEmptyMap(labels)
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: matchLabels,
	}

	deployment.Spec.MinReadySeconds = 0
	deployment.Spec.ProgressDeadlineSeconds = smoothoperatorutils.Pointer(int32(600))
	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
			MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 4},
		},
	}
	deployment.Spec.RevisionHistoryLimit = smoothoperatorutils.Pointer(int32(1))
	deployment.Spec.Replicas = smoothoperatorutils.Pointer(int32(2))

	podTemplateSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      matchLabels,
			Annotations: podTemplateAnnotations,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "socket", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapName}}},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:            "atom-generator",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"./atom"},
					Args:            []string{"-f=" + srvDir + "/config/" + configFileName, "-o=" + srvDir + "/data"},

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
							Name:          atomPortName,
							ContainerPort: atomPortNr,
						},
					},
					ImagePullPolicy: corev1.PullIfNotPresent,
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/index.xml",
								Port:   intstr.FromInt32(atomPortNr),
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
								Port:   intstr.FromInt32(atomPortNr),
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
	}

	podTemplateSpec.Spec.InitContainers[0].Image = r.AtomGeneratorImage
	podTemplateSpec.Spec.Containers[0].Image = r.LighttpdImage
	deployment.Spec.Template = podTemplateSpec

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, deployment, deployment); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, deployment, r.Scheme)

}

func getBareService(obj metav1.Object) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-atom",
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateService(atom *pdoknlv3.Atom, service *corev1.Service) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(atom.GetLabels())
	selector := smoothoperatorutils.CloneOrEmptyMap(atom.GetLabels())
	selector[appLabelKey] = atomName
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, service, labels); err != nil {
		return err
	}

	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:       atomPortName,
				Port:       atomPortNr,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt32(atomPortNr),
			},
		},
		Selector: selector,
	}
	if err := smoothoperatorutils.EnsureSetGVK(r.Client, service, service); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, service, r.Scheme)
}

func getBareStripPrefixMiddleware(obj metav1.Object) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + stripPrefixName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateStripPrefixMiddleware(atom *pdoknlv3.Atom, middleware *traefikiov1alpha1.Middleware) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(atom.GetLabels())
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		StripPrefix: &traefikdynamic.StripPrefix{
			Prefixes: []string{"/" + atom.GetBaseURLPath() + "/"}},
	}

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, middleware, r.Scheme)
}

func getBareCorsHeadersMiddleware(obj metav1.Object) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + corsHeadersName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
			UID:       obj.GetUID(),
		},
	}
}

func (r *AtomReconciler) mutateCorsHeadersMiddleware(atom *pdoknlv3.Atom, middleware *traefikiov1alpha1.Middleware) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(atom.GetLabels())
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		Headers: &traefikdynamic.Headers{
			CustomResponseHeaders: map[string]string{
				"Access-Control-Allow-Headers": "Content-Type",
				"Access-Control-Allow-Method":  "GET, HEAD, OPTIONS",
				"Access-Control-Allow-Origin":  "*",
			},
		},
	}
	middleware.Spec.Headers.FrameDeny = true
	if err := smoothoperatorutils.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}

	return ctrl.SetControllerReference(atom, middleware, r.Scheme)
}

func getBareDownloadLinkMiddleware(obj metav1.Object, index int8) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + downloadsName + "-" + strconv.Itoa(int(index)),
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateDownloadLinkMiddleware(atom *pdoknlv3.Atom, downloadLink *pdoknlv3.DownloadLink, middleware *traefikiov1alpha1.Middleware) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(atom.GetLabels())
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}

	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		ReplacePathRegex: &traefikdynamic.ReplacePathRegex{
			Regex:       getDownloadLinkRegex(atom, downloadLink),
			Replacement: getDownloadLinkReplacement(downloadLink),
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, middleware, r.Scheme)
}

func getBareIngressRoute(obj metav1.Object) *traefikiov1alpha1.IngressRoute {
	return &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateIngressRoute(atom *pdoknlv3.Atom, ingressRoute *traefikiov1alpha1.IngressRoute) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(atom.GetLabels())
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, ingressRoute, labels); err != nil {
		return err
	}

	ingressRoute.Spec = traefikiov1alpha1.IngressRouteSpec{
		Routes: []traefikiov1alpha1.Route{
			{
				Kind:  "Rule",
				Match: getMatchRuleForIndex(atom),
				Services: []traefikiov1alpha1.Service{
					{
						LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
							Name: getBareService(atom).GetName(),
							Kind: "Service",
							Port: intstr.FromInt32(atomPortNr),
						},
					},
				},
				Middlewares: []traefikiov1alpha1.MiddlewareRef{
					{
						Name:      atom.Name + "-" + stripPrefixName,
						Namespace: atom.GetNamespace(),
					},
					{
						Name:      atom.Name + "-" + corsHeadersName,
						Namespace: atom.GetNamespace(),
					},
				},
			},
		},
	}

	azureStorageRule := traefikiov1alpha1.Route{
		Kind:  "Rule",
		Match: getMatchRuleForDownloads(atom),
		Services: []traefikiov1alpha1.Service{
			{
				LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
					Name:           "azure-storage",
					Port:           intstr.IntOrString{Type: intstr.String, StrVal: "azure-storage"},
					PassHostHeader: smoothoperatorutils.Pointer(false),
					Kind:           "Service",
				},
			},
		},
		Middlewares: []traefikiov1alpha1.MiddlewareRef{
			{
				Name:      atom.Name + "-" + corsHeadersName,
				Namespace: atom.GetNamespace(),
			},
		},
	}
	// Set additional Azure storage middleware per download link
	for index := range atom.GetIndexedDownloadLinks() {
		middlewareRef := traefikiov1alpha1.MiddlewareRef{
			Name:      atom.Name + "-" + downloadsName + "-" + strconv.Itoa(int(index)),
			Namespace: atom.GetNamespace(),
		}
		azureStorageRule.Middlewares = append(azureStorageRule.Middlewares, middlewareRef)
	}
	ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, azureStorageRule)

	// Set additional routes per datasetFeed
	for _, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		matchRule := getMatchRuleForDatasetFeed(atom, &datasetFeed)
		rule := getDefaultRule(atom, matchRule)
		ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, rule)
	}

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, ingressRoute, ingressRoute); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, ingressRoute, r.Scheme)
}

func getBarePodDisruptionBudget(obj metav1.Object) *v1.PodDisruptionBudget {
	return &v1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-atom-pdb",
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutatePodDisruptionBudget(atom *pdoknlv3.Atom, podDisruptionBudget *v1.PodDisruptionBudget) error {
	labels := smoothoperatorutils.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = atomName
	if err := smoothoperatorutils.SetImmutableLabels(r.Client, podDisruptionBudget, labels); err != nil {
		return err
	}

	matchLabels := smoothoperatorutils.CloneOrEmptyMap(labels)
	podDisruptionBudget.Spec = v1.PodDisruptionBudgetSpec{
		MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Selector: &metav1.LabelSelector{
			MatchLabels: matchLabels,
		},
	}

	if err := smoothoperatorutils.EnsureSetGVK(r.Client, podDisruptionBudget, podDisruptionBudget); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, podDisruptionBudget, r.Scheme)
}

func getGeneratorConfig(atom *pdoknlv3.Atom, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	atomGeneratorConfig, err := generator.MapAtomV3ToAtomGeneratorConfig(*atom, *ownerInfo)
	if err != nil {
		return "", fmt.Errorf("failed to map the V3 atom to generator config: %w", err)
	}

	yamlConfig, err := yaml.Marshal(&atomGeneratorConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal the generator config to yaml: %w", err)
	}
	return string(yamlConfig), nil
}

func getMatchRuleForIndex(atom *pdoknlv3.Atom) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && Path(`/" + atom.GetBaseURLPath() + "/index.xml`)"
}

func getMatchRuleForDownloads(atom *pdoknlv3.Atom) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && PathPrefix(`/" + atom.GetBaseURLPath() + "/downloads/`)"
}

func getMatchRuleForDatasetFeed(atom *pdoknlv3.Atom, datasetFeed *pdoknlv3.DatasetFeed) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && Path(`/" + atom.GetBaseURLPath() + "/" + datasetFeed.TechnicalName + ".xml`)"
}

func getDefaultRule(atom *pdoknlv3.Atom, matchRule string) traefikiov1alpha1.Route {
	return traefikiov1alpha1.Route{
		Kind:  "Rule",
		Match: matchRule,
		Services: []traefikiov1alpha1.Service{
			{
				LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
					Name: getBareService(atom).GetName(),
					Kind: "Service",
					Port: intstr.FromInt32(atomPortNr),
				},
			},
		},
		Middlewares: []traefikiov1alpha1.MiddlewareRef{
			{
				Name:      atom.Name + "-" + stripPrefixName,
				Namespace: atom.GetNamespace(),
			},
			{
				Name:      atom.Name + "-" + corsHeadersName,
				Namespace: atom.GetNamespace(),
			},
		},
	}
}

func getDownloadLinkRegex(atom *pdoknlv3.Atom, downloadLink *pdoknlv3.DownloadLink) string {
	return fmt.Sprintf("^/%s/downloads/(%s)", atom.GetBaseURLPath(), downloadLink.GetBlobName())
}

func getDownloadLinkReplacement(downloadLink *pdoknlv3.DownloadLink) string {
	return "/" + downloadLink.GetBlobPrefix() + "/$1"
}

func (r *AtomReconciler) logAndUpdateStatusFinished(ctx context.Context, atom *pdoknlv3.Atom, operationResults map[string]controllerutil.OperationResult) {
	lgr := log.FromContext(ctx)
	lgr.Info("operation results", "results", operationResults)
	r.updateStatus(ctx, atom, []metav1.Condition{{
		Type:               reconciledConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             reconciledConditionReasonSucces,
		ObservedGeneration: atom.Generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}}, operationResults)
}

func (r *AtomReconciler) updateStatus(ctx context.Context, atom *pdoknlv3.Atom, conditions []metav1.Condition, operationResults map[string]controllerutil.OperationResult) {
	lgr := log.FromContext(ctx)
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(atom), atom); err != nil {
		log.FromContext(ctx).Error(err, "unable to update status")
		return
	}

	if atom.Status == nil {
		atom.Status = &smoothoperatormodel.OperatorStatus{}
	}

	changed := false
	for _, condition := range conditions {
		changed = meta.SetStatusCondition(&atom.Status.Conditions, condition) || changed
	}
	if !equality.Semantic.DeepEqual(atom.Status.OperationResults, operationResults) {
		atom.Status.OperationResults = operationResults
		changed = true
	}
	if !changed {
		return
	}
	if err := r.Status().Update(ctx, atom); err != nil {
		lgr.Error(err, "unable to update status")
	}
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
