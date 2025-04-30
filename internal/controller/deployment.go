package controller

import (
	v3 "github.com/pdok/atom-operator/api/v3"
	controller2 "github.com/pdok/smooth-operator/pkg/util"
	v2 "k8s.io/api/apps/v1"
	v4 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func getBareDeployment(obj v1.Object) *v2.Deployment {
	return &v2.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name: obj.GetName() + "-" + atomName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

//nolint:funlen
func (r *AtomReconciler) mutateDeployment(atom *v3.Atom, deployment *v2.Deployment, configMapName string) error {
	labels := controller2.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = atomName
	if err := controller2.SetImmutableLabels(r.Client, deployment, labels); err != nil {
		return err
	}

	podTemplateAnnotations := controller2.CloneOrEmptyMap(deployment.Spec.Template.GetAnnotations())

	matchLabels := controller2.CloneOrEmptyMap(labels)
	deployment.Spec.Selector = &v1.LabelSelector{
		MatchLabels: matchLabels,
	}

	deployment.Spec.MinReadySeconds = 0
	deployment.Spec.ProgressDeadlineSeconds = controller2.Pointer(int32(600))
	deployment.Spec.Strategy = v2.DeploymentStrategy{
		Type: v2.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &v2.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
			MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 4},
		},
	}
	deployment.Spec.RevisionHistoryLimit = controller2.Pointer(int32(1))
	deployment.Spec.Replicas = controller2.Pointer(int32(2))

	podTemplateSpec := v4.PodTemplateSpec{
		ObjectMeta: v1.ObjectMeta{
			Labels:      matchLabels,
			Annotations: podTemplateAnnotations,
		},
		Spec: v4.PodSpec{
			Volumes: []v4.Volume{
				{Name: "data", VolumeSource: v4.VolumeSource{EmptyDir: &v4.EmptyDirVolumeSource{}}},
				{Name: "socket", VolumeSource: v4.VolumeSource{EmptyDir: &v4.EmptyDirVolumeSource{}}},
				{Name: "config", VolumeSource: v4.VolumeSource{ConfigMap: &v4.ConfigMapVolumeSource{
					LocalObjectReference: v4.LocalObjectReference{Name: configMapName}}},
				},
			},
			InitContainers: []v4.Container{
				{
					Name:            "atom-generator",
					ImagePullPolicy: v4.PullIfNotPresent,
					Command:         []string{"./atom"},
					Args:            []string{"-f=" + srvDir + "/config/" + configFileName, "-o=" + srvDir + "/data"},

					VolumeMounts: []v4.VolumeMount{
						{Name: "data", MountPath: srvDir + "/data"},
						{Name: "config", MountPath: srvDir + "/config"},
					},
				},
			},
			Containers: []v4.Container{
				{
					Name: "atom-service",
					Ports: []v4.ContainerPort{
						{
							Name:          atomPortName,
							ContainerPort: atomPortNr,
						},
					},
					ImagePullPolicy: v4.PullIfNotPresent,
					LivenessProbe: &v4.Probe{
						ProbeHandler: v4.ProbeHandler{
							HTTPGet: &v4.HTTPGetAction{
								Path:   "/index.xml",
								Port:   intstr.FromInt32(atomPortNr),
								Scheme: v4.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 5,
						TimeoutSeconds:      5,
						PeriodSeconds:       10,
					},
					ReadinessProbe: &v4.Probe{
						ProbeHandler: v4.ProbeHandler{
							HTTPGet: &v4.HTTPGetAction{
								Path:   "/index.xml",
								Port:   intstr.FromInt32(atomPortNr),
								Scheme: v4.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 5,
						TimeoutSeconds:      5,
						PeriodSeconds:       10,
					},
					Resources: v4.ResourceRequirements{
						Limits: v4.ResourceList{
							v4.ResourceMemory: resource.MustParse("64M"),
						},
						Requests: v4.ResourceList{
							v4.ResourceCPU: resource.MustParse("0.01"),
						},
					},
					VolumeMounts: []v4.VolumeMount{
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

	if err := controller2.EnsureSetGVK(r.Client, deployment, deployment); err != nil {
		return err
	}
	return controllerruntime.SetControllerReference(atom, deployment, r.Scheme)

}
