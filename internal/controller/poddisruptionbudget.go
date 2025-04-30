package controller

import (
	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getBarePodDisruptionBudget(obj metav1.Object) *policyv1.PodDisruptionBudget {
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + "-atom-pdb",
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutatePodDisruptionBudget(atom *pdoknlv3.Atom, podDisruptionBudget *policyv1.PodDisruptionBudget) error {
	labels := smoothutil.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = atomName
	if err := smoothutil.SetImmutableLabels(r.Client, podDisruptionBudget, labels); err != nil {
		return err
	}

	matchLabels := smoothutil.CloneOrEmptyMap(labels)
	podDisruptionBudget.Spec = policyv1.PodDisruptionBudgetSpec{
		MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Selector: &metav1.LabelSelector{
			MatchLabels: matchLabels,
		},
	}

	if err := smoothutil.EnsureSetGVK(r.Client, podDisruptionBudget, podDisruptionBudget); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, podDisruptionBudget, r.Scheme)
}
