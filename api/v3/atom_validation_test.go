package v3

import (
	"fmt"
	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"os"
	"testing"
)

func TestValidateAtomWithoutClusterChecks(t *testing.T) {
	myWarnings := []string{}
	myAllErrors := field.ErrorList{}
	type args struct {
		warnings *[]string
		allErrs  *field.ErrorList
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
		{
			name: "v3-asis-no-error-no-warning",
			args: args{
				warnings: &myWarnings,
				allErrs:  &myAllErrors,
			},
		},
	}
	for _, tt := range tests {
		input, err := os.ReadFile("test_data/input/" + tt.name + ".yaml")
		if err != nil {
			t.Errorf("os.ReadFile() error = %v ", err)
		}
		atom := &Atom{}
		if err := yaml.Unmarshal(input, atom); err != nil {
			t.Errorf("yaml.Unmarshal() error = %v", err)
		}

		t.Run(tt.name, func(t *testing.T) {
			ValidateAtomWithoutClusterChecks(atom, tt.args.warnings, tt.args.allErrs)
			if tt.args.warnings != nil {
				fmt.Printf("tt.args.warnings: \n%v\n", tt.args.warnings)
			}
		})
	}
}
