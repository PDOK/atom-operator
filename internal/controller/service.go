package controller

import (
	v3 "github.com/pdok/atom-operator/api/v3"
	controller2 "github.com/pdok/smooth-operator/pkg/util"
	v2 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func getBareService(obj v1.Object) *v2.Service {
	return &v2.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      obj.GetName() + "-atom",
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateService(atom *v3.Atom, service *v2.Service) error {
	labels := controller2.CloneOrEmptyMap(atom.GetLabels())
	selector := controller2.CloneOrEmptyMap(atom.GetLabels())
	selector[appLabelKey] = atomName
	if err := controller2.SetImmutableLabels(r.Client, service, labels); err != nil {
		return err
	}

	service.Spec = v2.ServiceSpec{
		Ports: []v2.ServicePort{
			{
				Name:       atomPortName,
				Port:       atomPortNr,
				Protocol:   v2.ProtocolTCP,
				TargetPort: intstr.FromInt32(atomPortNr),
			},
		},
		Selector: selector,
	}
	if err := controller2.EnsureSetGVK(r.Client, service, service); err != nil {
		return err
	}
	return controllerruntime.SetControllerReference(atom, service, r.Scheme)
}
