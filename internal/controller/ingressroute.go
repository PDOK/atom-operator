package controller

import (
	"crypto/sha1" //nolint:gosec  // sha1 is only used for ID generation here, not crypto
	"fmt"
	"net/url"
	"strconv"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getBareIngressRoute(obj metav1.Object) *traefikiov1alpha1.IngressRoute {
	return &traefikiov1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName() + nameSuffix,
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateIngressRoute(atom *pdoknlv3.Atom, ingressRoute *traefikiov1alpha1.IngressRoute) error {
	labels := getLabels(atom)
	if err := smoothutil.SetImmutableLabels(r.Client, ingressRoute, labels); err != nil {
		return err
	}

	baseURL := atom.Spec.Service.BaseURL

	// TODO move to smoothoperator function
	ingressRoute.Annotations = map[string]string{
		"uptime.pdok.nl/id":   fmt.Sprintf("%x", sha1.Sum([]byte(atom.Name+nameSuffix))), //nolint:gosec  // sha1 is only used for ID generation here, not crypto
		"uptime.pdok.nl/name": atom.Spec.Service.Title + " ATOM",
		"uptime.pdok.nl/url":  baseURL.JoinPath("index.xml").String(),
		"uptime.pdok.nl/tags": "public-stats,atom",
	}

	ingressRoute.Spec = traefikiov1alpha1.IngressRouteSpec{
		Routes: []traefikiov1alpha1.Route{
			{
				Kind:  "Rule",
				Match: getMatchRule(baseURL.JoinPath("index.xml"), false),
				Services: []traefikiov1alpha1.Service{
					{
						LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
							Name: getBareService(atom).GetName(),
							Kind: "Service",
							Port: intstr.FromInt32(atomPortNr),
						},
					},
				},
				Middlewares: []traefikiov1alpha1.MiddlewareRef{
					{
						Name:      atom.Name + headersSuffix,
						Namespace: atom.GetNamespace(),
					},
					{
						Name:      atom.Name + stripPrefixSuffix,
						Namespace: atom.GetNamespace(),
					},
				},
			},
		},
	}

	// Set additional routes per datasetFeed
	for _, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		matchRule := getMatchRule(baseURL.JoinPath(datasetFeed.TechnicalName+".xml"), false)
		rule := getDefaultRule(atom, matchRule)
		ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, rule)
	}

	azureStorageRule := traefikiov1alpha1.Route{
		Kind:  "Rule",
		Match: getMatchRule(baseURL.JoinPath("downloads/"), true),
		Services: []traefikiov1alpha1.Service{
			{
				LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
					Name:           "azure-storage",
					Port:           intstr.IntOrString{Type: intstr.String, StrVal: "azure-storage"},
					PassHostHeader: smoothutil.Pointer(false),
					Kind:           "Service",
				},
			},
		},
		Middlewares: []traefikiov1alpha1.MiddlewareRef{
			{
				Name:      atom.Name + headersSuffix,
				Namespace: atom.GetNamespace(),
			},
		},
	}
	// Set additional Azure storage middleware per download link
	for _, group := range getDownloadLinkGroups(atom.GetDownloadLinks()) {
		middlewareRef := traefikiov1alpha1.MiddlewareRef{
			Name:      atom.Name + downloadsSuffix + strconv.Itoa(*group.index),
			Namespace: atom.GetNamespace(),
		}
		azureStorageRule.Middlewares = append(azureStorageRule.Middlewares, middlewareRef)
	}
	ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, azureStorageRule)

	if err := smoothutil.EnsureSetGVK(r.Client, ingressRoute, ingressRoute); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, ingressRoute, r.Scheme)
}

func getMatchRule(url *url.URL, pathPrefix bool) string {
	host := fmt.Sprintf("(Host(`localhost`) || Host(`%s`))", url.Hostname())
	pathType := "Path"
	if pathPrefix {
		pathType = "PathPrefix"
	}
	path := fmt.Sprintf("%s(`%s`)", pathType, url.Path)
	return fmt.Sprintf("%s && %s", host, path)
}

func getDefaultRule(atom *pdoknlv3.Atom, matchRule string) traefikiov1alpha1.Route {
	return traefikiov1alpha1.Route{
		Kind:  "Rule",
		Match: matchRule,
		Services: []traefikiov1alpha1.Service{
			{
				LoadBalancerSpec: traefikiov1alpha1.LoadBalancerSpec{
					Name: getBareService(atom).GetName(),
					Kind: "Service",
					Port: intstr.FromInt32(atomPortNr),
				},
			},
		},
		Middlewares: []traefikiov1alpha1.MiddlewareRef{
			{
				Name:      atom.Name + headersSuffix,
				Namespace: atom.GetNamespace(),
			},
			{
				Name:      atom.Name + stripPrefixSuffix,
				Namespace: atom.GetNamespace(),
			},
		},
	}
}
