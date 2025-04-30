package controller

import (
	"fmt"
	"strconv"

	v3 "github.com/pdok/atom-operator/api/v3"
	controller2 "github.com/pdok/smooth-operator/pkg/util"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func getBareStripPrefixMiddleware(obj v1.Object) *v1alpha1.Middleware {
	return &v1alpha1.Middleware{
		ObjectMeta: v1.ObjectMeta{
			Name: obj.GetName() + "-" + stripPrefixName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateStripPrefixMiddleware(atom *v3.Atom, middleware *v1alpha1.Middleware) error {
	labels := controller2.CloneOrEmptyMap(atom.GetLabels())
	if err := controller2.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = v1alpha1.MiddlewareSpec{
		StripPrefix: &dynamic.StripPrefix{
			Prefixes: []string{"/" + atom.GetBaseURLPath() + "/"}},
	}

	if err := controller2.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}
	return controllerruntime.SetControllerReference(atom, middleware, r.Scheme)
}

func getBareCorsHeadersMiddleware(obj v1.Object) *v1alpha1.Middleware {
	return &v1alpha1.Middleware{
		ObjectMeta: v1.ObjectMeta{
			Name: obj.GetName() + "-" + corsHeadersName,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
			UID:       obj.GetUID(),
		},
	}
}

func (r *AtomReconciler) mutateCorsHeadersMiddleware(atom *v3.Atom, middleware *v1alpha1.Middleware) error {
	labels := controller2.CloneOrEmptyMap(atom.GetLabels())
	if err := controller2.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = v1alpha1.MiddlewareSpec{
		Headers: &dynamic.Headers{
			CustomResponseHeaders: map[string]string{
				"Access-Control-Allow-Headers": "Content-Type",
				"Access-Control-Allow-Method":  "GET, HEAD, OPTIONS",
				"Access-Control-Allow-Origin":  "*",
			},
		},
	}
	middleware.Spec.Headers.FrameDeny = true
	if err := controller2.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}

	return controllerruntime.SetControllerReference(atom, middleware, r.Scheme)
}

func getBareDownloadLinkMiddleware(obj v1.Object, index int) *v1alpha1.Middleware {
	return &v1alpha1.Middleware{
		ObjectMeta: v1.ObjectMeta{
			Name: obj.GetName() + "-" + downloadsName + "-" + strconv.Itoa(index),
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateDownloadLinkMiddleware(atom *v3.Atom, downloadLink *v3.DownloadLink, middleware *v1alpha1.Middleware) error {
	labels := controller2.CloneOrEmptyMap(atom.GetLabels())
	if err := controller2.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}

	middleware.Spec = v1alpha1.MiddlewareSpec{
		ReplacePathRegex: &dynamic.ReplacePathRegex{
			Regex:       getDownloadLinkRegex(atom, downloadLink),
			Replacement: getDownloadLinkReplacement(downloadLink),
		},
	}

	if err := controller2.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}
	return controllerruntime.SetControllerReference(atom, middleware, r.Scheme)
}

func getDownloadLinkRegex(atom *v3.Atom, downloadLink *v3.DownloadLink) string {
	return fmt.Sprintf("^/%s/downloads/(%s)", atom.GetBaseURLPath(), downloadLink.GetBlobName())
}

func getDownloadLinkReplacement(downloadLink *v3.DownloadLink) string {
	return "/" + downloadLink.GetBlobPrefix() + "/$1"
}
