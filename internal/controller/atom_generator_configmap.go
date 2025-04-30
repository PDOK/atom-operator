package controller

import (
	"fmt"

	v3 "github.com/pdok/atom-operator/api/v3"
	"github.com/pdok/atom-operator/internal/controller/generator"
	v4 "github.com/pdok/smooth-operator/api/v1"
	controller2 "github.com/pdok/smooth-operator/pkg/util"
	v2 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func getBareConfigMap(obj v1.Object) *v2.ConfigMap {
	return &v2.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      getBareDeployment(obj).GetName(),
			Namespace: obj.GetNamespace(),
		},
	}
}

func (r *AtomReconciler) mutateAtomGeneratorConfigMap(atom *v3.Atom, ownerInfo *v4.OwnerInfo, configMap *v2.ConfigMap) error {
	labels := controller2.CloneOrEmptyMap(atom.GetLabels())
	labels[appLabelKey] = atomName
	if err := controller2.SetImmutableLabels(r.Client, configMap, labels); err != nil {
		return err
	}

	if len(configMap.Data) == 0 {
		generatorConfig, err := getGeneratorConfig(atom, ownerInfo)
		if err != nil {
			return err
		}
		configMap.Data = map[string]string{configFileName: generatorConfig}
	}
	configMap.Immutable = controller2.Pointer(true)

	if err := controller2.EnsureSetGVK(r.Client, configMap, configMap); err != nil {
		return err
	}
	if err := controllerruntime.SetControllerReference(atom, configMap, r.Scheme); err != nil {
		return err
	}
	return controller2.AddHashSuffix(configMap)
}

func getGeneratorConfig(atom *v3.Atom, ownerInfo *v4.OwnerInfo) (config string, err error) {
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
