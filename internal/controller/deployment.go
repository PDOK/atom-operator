package controller

import (
	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	evictAnnotation            = "cluster-autoscaler.kubernetes.io/safe-to-evict"
	evictValue                 = "true"
	defaultContainerAnnotation = "kubectl.kubernetes.io/default-container"
	versionCheckerAnnotation   = "priority.version-checker.io/atom-service"
	versionCheckerPriority     = "8"
)

func getBareDeployment(obj metav1.Object) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + appName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

//nolint:funlen
func (r *AtomReconciler) mutateDeployment(atom *pdoknlv3.Atom, deployment *appsv1.Deployment, configMapName string) error {
	labels := smoothutil.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = appName
	if err := smoothutil.SetImmutableLabels(r.Client, deployment, labels); err != nil {
		return err
	}

	podTemplateAnnotations := smoothutil.CloneOrEmptyMap(deployment.Spec.Template.GetAnnotations())
	podTemplateAnnotations[evictAnnotation] = evictValue
	podTemplateAnnotations[defaultContainerAnnotation] = "atom-service"
	podTemplateAnnotations[versionCheckerAnnotation] = versionCheckerPriority

	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}

	deployment.Spec.MinReadySeconds = 0
	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
			MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 4},
		},
	}
	deployment.Spec.RevisionHistoryLimit = smoothutil.Pointer(int32(1))
	deployment.Spec.Replicas = smoothutil.Pointer(int32(2))

	podTemplateSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      labels,
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
						{Name: "data", MountPath: "/var/www/"},
					},
				},
			},
		},
	}

	podTemplateSpec.Spec.InitContainers[0].Image = r.AtomGeneratorImage
	podTemplateSpec.Spec.Containers[0].Image = r.LighttpdImage
	deployment.Spec.Template = podTemplateSpec

	if err := smoothutil.EnsureSetGVK(r.Client, deployment, deployment); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, deployment, r.Scheme)

}
