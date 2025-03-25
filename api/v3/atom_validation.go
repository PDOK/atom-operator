package v3

import (
	"fmt"
	sharedValidation "github.com/pdok/smooth-operator/pkg/validation"
	"strings"
)

func (atom *Atom) ValidateCreate() ([]string, error) {
	warnings := []string{}
	reasons := []string{}

	err := sharedValidation.ValidateLabelsOnCreate(atom.Labels)
	if err != nil {
		reasons = append(reasons, fmt.Sprintf("%v", err))
	}

	validateAtom(atom, &warnings, &reasons)

	if len(reasons) > 0 {
		return warnings, fmt.Errorf("%s", strings.Join(reasons, ". "))
	} else {
		return warnings, nil
	}
}

func (atom *Atom) ValidateUpdate(atomOld *Atom) ([]string, error) {
	warnings := []string{}
	reasons := []string{}

	// Check labels did not change
	err := sharedValidation.ValidateLabelsOnUpdate(atomOld.Labels, atom.Labels)
	if err != nil {
		reasons = append(reasons, fmt.Sprintf("%v", err))
	}

	// Check service.baseURL did not change
	if atom.Spec.Service.BaseURL != atomOld.Spec.Service.BaseURL {
		reasons = append(reasons, fmt.Sprintf("service.baseURL is immutable"))
	}

	validateAtom(atom, &warnings, &reasons)

	if len(reasons) > 0 {
		return warnings, fmt.Errorf("%s", strings.Join(reasons, ". "))
	} else {
		return warnings, nil
	}
}

func validateAtom(atom *Atom, warnings *[]string, reasons *[]string) {
	if strings.Contains(atom.GetName(), "atom") {
		*warnings = append(*warnings, sharedValidation.FormatValidationWarning("name should not contain atom", atom.GroupVersionKind(), atom.GetName()))
	}

	for _, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		for _, entry := range datasetFeed.Entries {
			if linkCount := len(entry.DownloadLinks); linkCount > 1 && entry.Content == nil {
				*reasons = append(*reasons, "content is required for an Entry with more than 1 DownloadLink")
			}
		}
	}

	service := atom.Spec.Service

	err := sharedValidation.ValidateBaseURL(service.BaseURL)
	if err != nil {
		*reasons = append(*reasons, fmt.Sprintf("%v", err))
	}
}
