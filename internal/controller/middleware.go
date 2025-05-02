package controller

import (
	"fmt"
	"strconv"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getBareStripPrefixMiddleware(obj metav1.Object) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + "-" + stripPrefixSuffix,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateStripPrefixMiddleware(atom *pdoknlv3.Atom, middleware *traefikiov1alpha1.Middleware) error {
	labels := smoothutil.CloneOrEmptyMap(atom.GetLabels())
	if err := smoothutil.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		StripPrefix: &dynamic.StripPrefix{
			Prefixes: []string{"/" + atom.GetBaseURLPath() + "/"}},
	}

	if err := smoothutil.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, middleware, r.Scheme)
}

func getBareCorsHeadersMiddleware(obj metav1.Object) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + headersSuffix,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
			UID:       obj.GetUID(),
		},
	}
}

func (r *AtomReconciler) mutateCorsHeadersMiddleware(atom *pdoknlv3.Atom, middleware *traefikiov1alpha1.Middleware) error {
	labels := smoothutil.CloneOrEmptyMap(atom.GetLabels())
	if err := smoothutil.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		Headers: &dynamic.Headers{
			CustomResponseHeaders: map[string]string{
				"Access-Control-Allow-Headers": "Content-Type",
				"Access-Control-Allow-Method":  "GET, HEAD, OPTIONS",
				"Access-Control-Allow-Origin":  "*",
			},
		},
	}
	middleware.Spec.Headers.FrameDeny = true
	if err := smoothutil.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}

	return ctrl.SetControllerReference(atom, middleware, r.Scheme)
}

func getBareDownloadLinkMiddleware(obj metav1.Object, index int) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + downloadsSuffix + strconv.Itoa(index),
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateDownloadLinkMiddleware(atom *pdoknlv3.Atom, downloadLink *pdoknlv3.DownloadLink, middleware *traefikiov1alpha1.Middleware) error {
	labels := smoothutil.CloneOrEmptyMap(atom.GetLabels())
	if err := smoothutil.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}

	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		ReplacePathRegex: &dynamic.ReplacePathRegex{
			Regex:       getDownloadLinkRegex(atom, downloadLink),
			Replacement: getDownloadLinkReplacement(downloadLink),
		},
	}

	if err := smoothutil.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, middleware, r.Scheme)
}

func getDownloadLinkRegex(atom *pdoknlv3.Atom, downloadLink *pdoknlv3.DownloadLink) string {
	return fmt.Sprintf("^/%s/downloads/(%s)", atom.GetBaseURLPath(), downloadLink.GetBlobName())
}

func getDownloadLinkReplacement(downloadLink *pdoknlv3.DownloadLink) string {
	return "/" + downloadLink.GetBlobPrefix() + "/$1"
}
