package controller

import (
	"context"
	"time"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *AtomReconciler) logAndUpdateStatusError(ctx context.Context, atom *pdoknlv3.Atom, err error) {
	lgr := logf.FromContext(ctx)
	lgr.Error(err, "reconcile error")

	r.updateStatus(ctx, atom, []metav1.Condition{{
		Type:               reconciledConditionType,
		Status:             metav1.ConditionFalse,
		Reason:             reconciledConditionReasonError,
		Message:            err.Error(),
		ObservedGeneration: atom.Generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}}, nil)
}

func (r *AtomReconciler) logAndUpdateStatusFinished(ctx context.Context, atom *pdoknlv3.Atom, operationResults map[string]controllerutil.OperationResult) {
	lgr := logf.FromContext(ctx)
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
	lgr := logf.FromContext(ctx)
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(atom), atom); err != nil {
		logf.FromContext(ctx).Error(err, "unable to update status")
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
