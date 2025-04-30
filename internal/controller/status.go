package controller

import (
	"context"
	"time"

	v3 "github.com/pdok/atom-operator/api/v3"
	"github.com/pdok/smooth-operator/model"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *AtomReconciler) logAndUpdateStatusError(ctx context.Context, atom *v3.Atom, err error) {
	r.updateStatus(ctx, atom, []v1.Condition{{
		Type:               reconciledConditionType,
		Status:             v1.ConditionFalse,
		Reason:             reconciledConditionReasonError,
		Message:            err.Error(),
		ObservedGeneration: atom.Generation,
		LastTransitionTime: v1.NewTime(time.Now()),
	}}, nil)
}

func (r *AtomReconciler) logAndUpdateStatusFinished(ctx context.Context, atom *v3.Atom, operationResults map[string]controllerutil.OperationResult) {
	lgr := log.FromContext(ctx)
	lgr.Info("operation results", "results", operationResults)
	r.updateStatus(ctx, atom, []v1.Condition{{
		Type:               reconciledConditionType,
		Status:             v1.ConditionTrue,
		Reason:             reconciledConditionReasonSucces,
		ObservedGeneration: atom.Generation,
		LastTransitionTime: v1.NewTime(time.Now()),
	}}, operationResults)
}

func (r *AtomReconciler) updateStatus(ctx context.Context, atom *v3.Atom, conditions []v1.Condition, operationResults map[string]controllerutil.OperationResult) {
	lgr := log.FromContext(ctx)
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(atom), atom); err != nil {
		log.FromContext(ctx).Error(err, "unable to update status")
		return
	}

	if atom.Status == nil {
		atom.Status = &model.OperatorStatus{}
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
