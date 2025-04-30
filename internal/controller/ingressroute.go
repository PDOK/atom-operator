package controller

import (
	"strconv"

	v3 "github.com/pdok/atom-operator/api/v3"
	controller2 "github.com/pdok/smooth-operator/pkg/util"
	"github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func getBareIngressRoute(obj v1.Object) *v1alpha1.IngressRoute {
	return &v1alpha1.IngressRoute{
		ObjectMeta: v1.ObjectMeta{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateIngressRoute(atom *v3.Atom, ingressRoute *v1alpha1.IngressRoute) error {
	labels := controller2.CloneOrEmptyMap(atom.GetLabels())
	if err := controller2.SetImmutableLabels(r.Client, ingressRoute, labels); err != nil {
		return err
	}

	ingressRoute.Spec = v1alpha1.IngressRouteSpec{
		Routes: []v1alpha1.Route{
			{
				Kind:  "Rule",
				Match: getMatchRuleForIndex(atom),
				Services: []v1alpha1.Service{
					{
						LoadBalancerSpec: v1alpha1.LoadBalancerSpec{
							Name: getBareService(atom).GetName(),
							Kind: "Service",
							Port: intstr.FromInt32(atomPortNr),
						},
					},
				},
				Middlewares: []v1alpha1.MiddlewareRef{
					{
						Name:      atom.Name + "-" + stripPrefixName,
						Namespace: atom.GetNamespace(),
					},
					{
						Name:      atom.Name + "-" + corsHeadersName,
						Namespace: atom.GetNamespace(),
					},
				},
			},
		},
	}

	azureStorageRule := v1alpha1.Route{
		Kind:  "Rule",
		Match: getMatchRuleForDownloads(atom),
		Services: []v1alpha1.Service{
			{
				LoadBalancerSpec: v1alpha1.LoadBalancerSpec{
					Name:           "azure-storage",
					Port:           intstr.IntOrString{Type: intstr.String, StrVal: "azure-storage"},
					PassHostHeader: controller2.Pointer(false),
					Kind:           "Service",
				},
			},
		},
		Middlewares: []v1alpha1.MiddlewareRef{
			{
				Name:      atom.Name + "-" + corsHeadersName,
				Namespace: atom.GetNamespace(),
			},
		},
	}
	// Set additional Azure storage middleware per download link
	for index := range atom.GetDownloadLinks() {
		middlewareRef := v1alpha1.MiddlewareRef{
			Name:      atom.Name + "-" + downloadsName + "-" + strconv.Itoa(index),
			Namespace: atom.GetNamespace(),
		}
		azureStorageRule.Middlewares = append(azureStorageRule.Middlewares, middlewareRef)
	}
	ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, azureStorageRule)

	// Set additional routes per datasetFeed
	for _, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		matchRule := getMatchRuleForDatasetFeed(atom, &datasetFeed)
		rule := getDefaultRule(atom, matchRule)
		ingressRoute.Spec.Routes = append(ingressRoute.Spec.Routes, rule)
	}

	if err := controller2.EnsureSetGVK(r.Client, ingressRoute, ingressRoute); err != nil {
		return err
	}
	return controllerruntime.SetControllerReference(atom, ingressRoute, r.Scheme)
}

func getMatchRuleForIndex(atom *v3.Atom) string {
	return "Host(`" + v3.GetHost() + "`) && Path(`/" + atom.GetBaseURLPath() + "/index.xml`)"
}

func getMatchRuleForDownloads(atom *v3.Atom) string {
	return "Host(`" + v3.GetHost() + "`) && PathPrefix(`/" + atom.GetBaseURLPath() + "/downloads/`)"
}

func getMatchRuleForDatasetFeed(atom *v3.Atom, datasetFeed *v3.DatasetFeed) string {
	return "Host(`" + v3.GetHost() + "`) && Path(`/" + atom.GetBaseURLPath() + "/" + datasetFeed.TechnicalName + ".xml`)"
}

func getDefaultRule(atom *v3.Atom, matchRule string) v1alpha1.Route {
	return v1alpha1.Route{
		Kind:  "Rule",
		Match: matchRule,
		Services: []v1alpha1.Service{
			{
				LoadBalancerSpec: v1alpha1.LoadBalancerSpec{
					Name: getBareService(atom).GetName(),
					Kind: "Service",
					Port: intstr.FromInt32(atomPortNr),
				},
			},
		},
		Middlewares: []v1alpha1.MiddlewareRef{
			{
				Name:      atom.Name + "-" + stripPrefixName,
				Namespace: atom.GetNamespace(),
			},
			{
				Name:      atom.Name + "-" + corsHeadersName,
				Namespace: atom.GetNamespace(),
			},
		},
	}
}
