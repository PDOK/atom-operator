package v3

import (
	"fmt"

	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorvalidation "github.com/pdok/smooth-operator/pkg/validation"

	"strings"

	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (atom *Atom) ValidateCreate(c client.Client) ([]string, error) {
	warnings := []string{}
	reasons := []string{}

	err := smoothoperatorvalidation.ValidateLabelsOnCreate(atom.Labels)
	if err != nil {
		reasons = append(reasons, fmt.Sprintf("%v", err))
	}

	ValidateAtom(c, atom, &warnings, &reasons)

	if len(reasons) > 0 {
		return warnings, fmt.Errorf("%s", strings.Join(reasons, ". "))
	}

	return warnings, nil
}

func (atom *Atom) ValidateUpdate(c client.Client, atomOld *Atom) ([]string, error) {
	warnings := []string{}
	reasons := []string{}

	// Check labels did not change
	err := smoothoperatorvalidation.ValidateLabelsOnUpdate(atomOld.Labels, atom.Labels)
	if err != nil {
		reasons = append(reasons, fmt.Sprintf("%v", err))
	}

	// Check service.baseURL did not change
	if atom.Spec.Service.BaseURL != atomOld.Spec.Service.BaseURL {
		reasons = append(reasons, fmt.Sprintf("service.baseURL is immutable, oldBaseUrl: %s, newBaseUrl: %s", atomOld.Spec.Service.BaseURL, atom.Spec.Service.BaseURL))
	}

	ValidateAtom(c, atom, &warnings, &reasons)

	if len(reasons) > 0 {
		return warnings, fmt.Errorf("%s", strings.Join(reasons, ". "))
	}

	return warnings, nil
}

func ValidateAtom(c client.Client, atom *Atom, warnings *[]string, reasons *[]string) {
	ValidateAtomWithoutClusterChecks(atom, warnings, reasons)

	ownerInfo := &smoothoperatorv1.OwnerInfo{}
	objectKey := client.ObjectKey{
		Namespace: atom.Namespace,
		Name:      atom.Spec.Service.OwnerInfoRef,
	}
	ctx := context.Background()
	err := c.Get(ctx, objectKey, ownerInfo)
	if err != nil {
		*reasons = append(*reasons, fmt.Sprintf("%v", err))
	}

	if ownerInfo.Spec.Atom == nil {
		*reasons = append(*reasons, "no atom settings in ownerInfo: "+ownerInfo.Name)
	}
}

func ValidateAtomWithoutClusterChecks(atom *Atom, warnings *[]string, reasons *[]string) {
	if strings.Contains(atom.GetName(), "atom") {
		*warnings = append(*warnings, smoothoperatorvalidation.FormatValidationWarning("name should not contain atom", atom.GroupVersionKind(), atom.GetName()))
	}

	for _, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		for _, entry := range datasetFeed.Entries {
			if linkCount := len(entry.DownloadLinks); linkCount > 1 && entry.Content == nil {
				*reasons = append(*reasons, "content is required for an Entry with more than 1 DownloadLink")
			}
		}
	}

	service := atom.Spec.Service
	err := smoothoperatorvalidation.ValidateBaseURL(service.BaseURL)
	if err != nil {
		*reasons = append(*reasons, fmt.Sprintf("%v", err))
	}
}
