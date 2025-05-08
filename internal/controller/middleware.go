package controller

import (
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
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
			Prefixes: []string{atom.GetBaseUrl().Path + "/"}},
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

func (r *AtomReconciler) mutateHeadersMiddleware(atom *pdoknlv3.Atom, middleware *traefikiov1alpha1.Middleware, csp string) error {
	labels := getLabels(atom)
	if err := smoothutil.SetImmutableLabels(r.Client, middleware, labels); err != nil {
		return err
	}
	middleware.Spec = traefikiov1alpha1.MiddlewareSpec{
		Headers: &dynamic.Headers{
			// CSP
			ContentSecurityPolicy: csp,
			// Frame-Options
			FrameDeny: true,
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

	baseURL := atom.GetBaseUrl()

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

func getDownloadLinkRegex(url smoothoperatormodel.URL, files []string) string {
	return "^" + url.JoinPath("downloads", "("+strings.Join(files, "|")+")").Path
}

func getDownloadLinkGroups(links []pdoknlv3.DownloadLink) map[string]struct {
	index *int
	files []string
} {
	result := make(map[string]struct {
		index *int
		files []string
	})

	counter := 0

	for _, link := range links {
		prefix := link.GetBlobPrefix()
		file := link.GetBlobName()
		val, ok := result[prefix]
		if ok {
			if val.index == nil {
				index := counter
				val.index = &index
				counter++
			}
			val.files = append(val.files, file)
			result[prefix] = val
		} else {
			index := counter
			counter++
			result[prefix] = struct {
				index *int
				files []string
			}{index: &index, files: []string{file}}
		}
	}

	return result
}
