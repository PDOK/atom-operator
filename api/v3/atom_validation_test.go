package v3

import (
	//nolint:revive // ginkgo bdd
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	"k8s.io/apimachinery/pkg/util/validation/field"
	//"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"testing"
)

var (
	cfg          *rest.Config
	testTheme          = "TEST_THEME"
	TestTTLInt32 int32 = 30
)

func getFilledAtomv3() *Atom {

	return &Atom{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Atom",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "v3-asis-no-error-no-warning",
			Labels: map[string]string{
				"dataset-owner": "test_datasetowner",
				"dataset":       "test_dataset",
				"theme":         testTheme,
				"service-type":  "test_servicetype",
			},
		},
		Spec: AtomSpec{
			Lifecycle: &smoothoperatormodel.Lifecycle{
				TTLInDays: &TestTTLInt32,
			},
		},
	}
}

func TestValidateAtomWithoutClusterChecks(t *testing.T) {
	myWarnings := []string{}
	myAllErrors := field.ErrorList{}
	type args struct {
		atom     *Atom
		warnings *[]string
		allErrs  *field.ErrorList
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
		{
			name: "MyFirstTest",
			args: args{
				atom:     getFilledAtomv3(),
				warnings: &myWarnings,
				allErrs:  &myAllErrors,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ValidateAtomWithoutClusterChecks(tt.args.atom, tt.args.warnings, tt.args.allErrs)
		})
	}
}
