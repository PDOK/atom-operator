package controller

import (
	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getBareService(obj metav1.Object) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + nameSuffix,
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateService(atom *pdoknlv3.Atom, service *corev1.Service) error {
	labels := getLabels(atom)
	selector := labels
	if err := smoothutil.SetImmutableLabels(r.Client, service, labels); err != nil {
		return err
	}

	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:     atomPortName,
				Port:     atomPortNr,
				Protocol: corev1.ProtocolTCP,
			},
		},
		Selector: selector,
	}
	if err := smoothutil.EnsureSetGVK(r.Client, service, service); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, service, r.Scheme)
}
