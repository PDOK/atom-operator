package controller

import (
	v3 "github.com/pdok/atom-operator/api/v3"
	controller2 "github.com/pdok/smooth-operator/pkg/util"
	v2 "k8s.io/api/policy/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func getBarePodDisruptionBudget(obj v1.Object) *v2.PodDisruptionBudget {
	return &v2.PodDisruptionBudget{
		ObjectMeta: v1.ObjectMeta{
			Name:      obj.GetName() + "-atom-pdb",
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutatePodDisruptionBudget(atom *v3.Atom, podDisruptionBudget *v2.PodDisruptionBudget) error {
	labels := controller2.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = atomName
	if err := controller2.SetImmutableLabels(r.Client, podDisruptionBudget, labels); err != nil {
		return err
	}

	matchLabels := controller2.CloneOrEmptyMap(labels)
	podDisruptionBudget.Spec = v2.PodDisruptionBudgetSpec{
		MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Selector: &v1.LabelSelector{
			MatchLabels: matchLabels,
		},
	}

	if err := controller2.EnsureSetGVK(r.Client, podDisruptionBudget, podDisruptionBudget); err != nil {
		return err
	}
	return controllerruntime.SetControllerReference(atom, podDisruptionBudget, r.Scheme)
}
