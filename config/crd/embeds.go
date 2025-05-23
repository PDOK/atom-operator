package crd

//nolint:goimports
import (
	_ "embed"
	"github.com/pdok/smooth-operator/pkg/validation"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

//go:embed bases/pdok.nl_atoms.yaml
var atomCRD []byte

func init() {
	crd := v1.CustomResourceDefinition{}
	err := yaml.Unmarshal(atomCRD, &crd)
	if err != nil {
		panic(err)
	}

	err = validation.AddValidator(crd)
	if err != nil {
		panic(err)
	}
}
