package controller

import (
	"fmt"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	"github.com/pdok/atom-operator/internal/controller/generator"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func getBareConfigMap(obj metav1.Object) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getBareDeployment(obj).GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateAtomGeneratorConfigMap(atom *pdoknlv3.Atom, ownerInfo *smoothoperatorv1.OwnerInfo, configMap *corev1.ConfigMap) error {
	labels := smoothutil.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = atomName
	if err := smoothutil.SetImmutableLabels(r.Client, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		generatorConfig, err := getGeneratorConfig(atom, ownerInfo)
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{configFileName: generatorConfig}
	}
	configMap.Immutable = smoothutil.Pointer(true)

	if err := smoothutil.EnsureSetGVK(r.Client, configMap, configMap); err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(atom, configMap, r.Scheme); err != nil {
		return err
	}
	return smoothutil.AddHashSuffix(configMap)
}

func getGeneratorConfig(atom *pdoknlv3.Atom, ownerInfo *smoothoperatorv1.OwnerInfo) (config string, err error) {
	atomGeneratorConfig, err := generator.MapAtomV3ToAtomGeneratorConfig(*atom, *ownerInfo)
	if err != nil {
		return "", fmt.Errorf("failed to map the V3 atom to generator config: %w", err)
	}

	yamlConfig, err := yaml.Marshal(&atomGeneratorConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal the generator config to yaml: %w", err)
	}
	return string(yamlConfig), nil
}
