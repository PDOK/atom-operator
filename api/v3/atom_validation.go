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

	smoothoperatorvalidation.CheckBaseUrlImmutability(atomOld, atom, &reasons)

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
	var path string
	if strings.Contains(atom.GetName(), "atom") {
		path = "metadata.name"
		smoothoperatorvalidation.AddWarning(warnings, path, "should not contain atom", atom.GroupVersionKind(), atom.GetName())
	}

	for i, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		path = fmt.Sprintf("spec.service.datasetFeeds[%d]", i)
		if datasetFeed.DatasetMetadataLinks != nil && atom.Spec.Service.ServiceMetadataLinks != nil {
			if datasetFeed.DatasetMetadataLinks.MetadataIdentifier == atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier {
				smoothoperatorvalidation.AddReason(reasons, path+".datasetMetadataLinks.MetadataIdentifier", "should not be the same as spec.service.serviceMetadataLinks.metadataIdentifier")
			}
		}

		if datasetFeed.DatasetMetadataLinks != nil && datasetFeed.SpatialDatasetIdentifierCode == nil {
			smoothoperatorvalidation.AddReason(reasons, path+".spatialDatasetIdentifierCode", fmt.Sprintf("is required when %s is set", path+".datasetMetadataLinks"))
		}

		if datasetFeed.SpatialDatasetIdentifierCode != nil && datasetFeed.SpatialDatasetIdentifierNamespace == nil {
			smoothoperatorvalidation.AddReason(reasons, path+".spatialDatasetIdentifierNamespace", fmt.Sprintf("is required when %s is set", path+".spatialDatasetIdentifierCode"))
		}

		for in, entry := range datasetFeed.Entries {
			path = fmt.Sprintf("%s.entries[%d]", path, in)
			if linkCount := len(entry.DownloadLinks); linkCount > 1 && entry.Content == nil {
				smoothoperatorvalidation.AddReason(reasons, path+".spatialDatasetIdentifierNamespace", fmt.Sprintf("is required when there are 2 or more downloadLinks"))
			}
		}
	}
}
