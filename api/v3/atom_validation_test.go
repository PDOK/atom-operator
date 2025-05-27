package v3

import (
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"os"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestValidateAtomWithoutClusterChecks(t *testing.T) {

	tests := []struct {
		name             string
		expectedWarnings *[]string
		expectedErrors   *field.ErrorList
	}{
		// Lijst van testcases
		{
			name:             "no-error-no-warning",
			expectedWarnings: &[]string{},
			expectedErrors:   &field.ErrorList{},
		},
		{
			name:             "no-error-atom-name-warning",
			expectedWarnings: &[]string{"pdok.nl/v3, Kind=Atom/asis-readonly-prod-atom: metadata.name: should not contain atom"},
			expectedErrors:   &field.ErrorList{},
		},
		{
			name:             "no-error-tag-warning",
			expectedWarnings: &[]string{"pdok.nl/v3, Kind=Atom/with-theme-warning: metadata.labels.pdok.nl/tag: general.theme field is not supposed to be set"},
			expectedErrors:   &field.ErrorList{},
		},
	}
	for _, tt := range tests {
		actualWarnings := []string{}
		actualAllErrors := field.ErrorList{}
		input, err := os.ReadFile("test_data/input/" + tt.name + ".yaml")
		if err != nil {
			t.Errorf("os.ReadFile() error = %v ", err)
		}
		atom := &Atom{}
		if err := yaml.Unmarshal(input, atom); err != nil {
			t.Errorf("yaml.Unmarshal() error = %v", err)
		}

		t.Run(tt.name, func(t *testing.T) {
			ValidateAtomWithoutClusterChecks(atom, &actualWarnings, &actualAllErrors)
			diffWarnings := cmp.Diff(tt.expectedWarnings, &actualWarnings)
			if diffWarnings != "" {
				t.Errorf("Testing validation has different warnings: \n%v\n%v\n%v", diffWarnings, tt.expectedWarnings, actualWarnings)
			}
			diffErrors := cmp.Diff(tt.expectedErrors, &actualAllErrors)
			if diffErrors != "" {
				t.Errorf("Testing validation has different errors: \n%v\n%v\n%v", diffErrors, tt.expectedErrors, actualAllErrors)
			}
		})
	}
}
