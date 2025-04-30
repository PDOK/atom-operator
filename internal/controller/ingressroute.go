package controller

import (
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
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateIngressRoute(atom *pdoknlv3.Atom, ingressRoute *traefikiov1alpha1.IngressRoute) error {
	labels := smoothutil.CloneOrEmptyMap(atom.GetLabels())
	if err := smoothutil.SetImmutableLabels(r.Client, ingressRoute, labels); err != nil {
		return err
	}

	ingressRoute.Spec = traefikiov1alpha1.IngressRouteSpec{
		Routes: []traefikiov1alpha1.Route{
			{
				Kind:  "Rule",
				Match: getMatchRuleForIndex(atom),
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

	azureStorageRule := traefikiov1alpha1.Route{
		Kind:  "Rule",
		Match: getMatchRuleForDownloads(atom),
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
				Name:      atom.Name + "-" + corsHeadersName,
				Namespace: atom.GetNamespace(),
			},
		},
	}
	// Set additional Azure storage middleware per download link
	for index := range atom.GetDownloadLinks() {
		middlewareRef := traefikiov1alpha1.MiddlewareRef{
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

	if err := smoothutil.EnsureSetGVK(r.Client, ingressRoute, ingressRoute); err != nil {
		return err
	}
	return ctrl.SetControllerReference(atom, ingressRoute, r.Scheme)
}

func getMatchRuleForIndex(atom *pdoknlv3.Atom) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && Path(`/" + atom.GetBaseURLPath() + "/index.xml`)"
}

func getMatchRuleForDownloads(atom *pdoknlv3.Atom) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && PathPrefix(`/" + atom.GetBaseURLPath() + "/downloads/`)"
}

func getMatchRuleForDatasetFeed(atom *pdoknlv3.Atom, datasetFeed *pdoknlv3.DatasetFeed) string {
	return "Host(`" + pdoknlv3.GetHost() + "`) && Path(`/" + atom.GetBaseURLPath() + "/" + datasetFeed.TechnicalName + ".xml`)"
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
