package controller

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

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
			Name: obj.GetName() + stripPrefixSuffix,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateStripPrefixMiddleware(atom *pdoknlv3.Atom, middleware *traefikiov1alpha1.Middleware) error {
	labels := getLabels(atom)
	if err := smoothutil.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		StripPrefix: &dynamic.StripPrefix{
			Prefixes: []string{atom.GetBaseURL().Path}},
	}

	if err := smoothutil.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, middleware, r.Scheme)
}

func getBareHeadersMiddleware(obj metav1.Object) *traefikiov1alpha1.Middleware {
	return &traefikiov1alpha1.Middleware{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.GetName() + headersSuffix,
			// name might become too long. not handling here. will just fail on apply.
			Namespace: obj.GetNamespace(),
			UID:       obj.GetUID(),
		},
	}
}

func (r *AtomReconciler) mutateHeadersMiddleware(atom *pdoknlv3.Atom, middleware *traefikiov1alpha1.Middleware) error {
	labels := getLabels(atom)
	if err := smoothutil.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		Headers: &dynamic.Headers{
			CustomResponseHeaders: map[string]string{
				"Access-Control-Allow-Headers": "Content-Type",
				"Access-Control-Allow-Method":  "GET, OPTIONS, HEAD",
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

func (r *AtomReconciler) mutateDownloadLinkMiddleware(atom *pdoknlv3.Atom, prefix string, files []string, middleware *traefikiov1alpha1.Middleware) error {
	labels := getLabels(atom)
	if err := smoothutil.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}

	baseURL := atom.GetBaseURL()

	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		ReplacePathRegex: &dynamic.ReplacePathRegex{
			Regex:       getDownloadLinkRegex(baseURL, files),
			Replacement: "/" + prefix + "/$1",
		},
	}

	if err := smoothutil.EnsureSetGVK(r.Client, middleware, middleware); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, middleware, r.Scheme)
}

func getDownloadLinkRegex(baseURL url.URL, files []string) string {
	return fmt.Sprintf("^%sdownloads/(%s)", baseURL.Path, strings.Join(files, "|"))
}

func getDownloadLinkGroups(links []pdoknlv3.DownloadLink) []struct {
	prefix string
	files  []string
} {
	var temp map[string][]string

	for _, link := range links {
		temp[link.GetBlobPrefix()] = append(temp[link.GetBlobPrefix()], link.GetBlobName())
	}

	var result []struct {
		prefix string
		files  []string
	}

	for prefix, files := range temp {
		result = append(result, struct {
			prefix string
			files  []string
		}{prefix: prefix, files: files})
	}

	return result
}
