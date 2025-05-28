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
			name:             "1-no-error-no-warning",
			expectedWarnings: &[]string{},
			expectedErrors:   &field.ErrorList{},
		},
		{
			name:             "2-warning-atom-name",
			expectedWarnings: &[]string{"pdok.nl/v3, Kind=Atom/asis-readonly-prod-atom: metadata.name: should not contain atom"},
			expectedErrors:   &field.ErrorList{},
		},
		{
			name:             "3-error-dataset-metadata-id",
			expectedWarnings: &[]string{},
			expectedErrors: &field.ErrorList{&field.Error{
				Type:     "FieldValueInvalid",
				Field:    "spec.service.datasetFeeds[0].datasetMetadataLinks.metadataIdentifier",
				BadValue: "2751ba40-5100-4186-81be-b7fdee95b49c",
				Detail:   "should not be the same as spec.service.serviceMetadataLinks.metadataIdentifier",
			}},
		},
		{
			name:             "4-spatialDatasetIdentifierCode-missing-error",
			expectedWarnings: &[]string{},
			expectedErrors: &field.ErrorList{&field.Error{
				Type:     "FieldValueRequired",
				Field:    "spec.service.datasetFeeds[0].spatialDatasetIdentifierCode",
				BadValue: "",
				Detail:   "when spec.service.datasetFeeds[0].datasetMetadataLinks exists",
			}},
		},
		{
			name:             "5-spatialDatasetIdentifierNamespace-missing-error",
			expectedWarnings: &[]string{},
			expectedErrors: &field.ErrorList{&field.Error{
				Type:     "FieldValueRequired",
				Field:    "spec.service.datasetFeeds[0].spatialDatasetIdentifierNamespace",
				BadValue: "",
				Detail:   "when spec.service.datasetFeeds[0].spatialDatasetIdentifierCode exists",
			}},
		},
		{
			name:             "6-entry-content-missing-error",
			expectedWarnings: &[]string{},
			expectedErrors: &field.ErrorList{&field.Error{
				Type:     "FieldValueRequired",
				Field:    "spec.service.datasetFeeds[0].entries[0].content",
				BadValue: "",
				Detail:   "when spec.service.datasetFeeds[0].entries[0].downloadlinks has 2 or more elements",
			}},
		},
		{
			name:             "7-duplicate-entry-tech-name-error",
			expectedWarnings: &[]string{},
			expectedErrors: &field.ErrorList{&field.Error{
				Type:     "FieldValueDuplicate",
				Field:    "spec.service.datasetFeeds[0].entries[0].entries[1].technicalName",
				BadValue: "wetlands",
				Detail:   "",
			}},
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
