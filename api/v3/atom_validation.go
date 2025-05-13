package v3

import (
	"fmt"
	"slices"

	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorvalidation "github.com/pdok/smooth-operator/pkg/validation"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"strings"

	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (atom *Atom) ValidateCreate(c client.Client) ([]string, error) {
	var warnings []string
	var allErrs field.ErrorList

	err := smoothoperatorvalidation.ValidateLabelsOnCreate(atom.Labels)
	if err != nil {
		allErrs = append(allErrs, err)
	}

	ValidateAtom(c, atom, &warnings, &allErrs)

	if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, apierrors.NewInvalid(
		schema.GroupKind{Group: "pdok.nl", Kind: "Atom"},
		atom.Name, allErrs)
}

func (atom *Atom) ValidateUpdate(c client.Client, atomOld *Atom) ([]string, error) {
	var warnings []string
	var allErrs field.ErrorList
	smoothoperatorvalidation.ValidateLabelsOnUpdate(atomOld.Labels, atom.Labels, &allErrs)

	smoothoperatorvalidation.CheckBaseUrlImmutability(atomOld, atom, &allErrs)

	ValidateAtom(c, atom, &warnings, &allErrs)

	if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, apierrors.NewInvalid(
		schema.GroupKind{Group: "pdok.nl", Kind: "Atom"},
		atom.Name, allErrs)
}

func ValidateAtom(c client.Client, atom *Atom, warnings *[]string, allErrs *field.ErrorList) {
	ValidateAtomWithoutClusterChecks(atom, warnings, allErrs)

	ownerInfoRef := atom.Spec.Service.OwnerInfoRef
	ownerInfo := &smoothoperatorv1.OwnerInfo{}
	objectKey := client.ObjectKey{
		Namespace: atom.Namespace,
		Name:      ownerInfoRef,
	}
	ctx := context.Background()
	err := c.Get(ctx, objectKey, ownerInfo)
	fieldPath := field.NewPath("spec").Child("service").Child("ownerInfoRef")
	if err != nil {
		*allErrs = append(*allErrs, field.NotFound(fieldPath, ownerInfoRef))
		return
	}

	if ownerInfo.Spec.Atom == nil {
		*allErrs = append(*allErrs, field.Required(fieldPath, "spec.Atom missing in "+ownerInfo.Name))
	}
}

func ValidateAtomWithoutClusterChecks(atom *Atom, warnings *[]string, allErrs *field.ErrorList) {
	var fieldPath *field.Path
	if strings.Contains(atom.GetName(), "atom") {
		fieldPath = field.NewPath("metadata").Child("name")
		smoothoperatorvalidation.AddWarning(warnings, *fieldPath, "should not contain atom", atom.GroupVersionKind(), atom.GetName())
	}
	var feedNames []string
	for i, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		fieldPath = field.NewPath("spec").Child("service").Child("datasetFeeds").Index(i)

		if slices.Contains(feedNames, datasetFeed.TechnicalName) {
			*allErrs = append(*allErrs, field.Duplicate(fieldPath.Child("technicalName"), datasetFeed.TechnicalName))
		}

		feedNames = append(feedNames, datasetFeed.TechnicalName)

		if datasetFeed.DatasetMetadataLinks != nil && atom.Spec.Service.ServiceMetadataLinks != nil {
			if datasetFeed.DatasetMetadataLinks.MetadataIdentifier == atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier {
				*allErrs = append(*allErrs, field.Invalid(
					fieldPath.Child("datasetMetadataLinks").Child("metadataIdentifier"),
					datasetFeed.DatasetMetadataLinks.MetadataIdentifier,
					fmt.Sprintf("should not be the same as %s", field.NewPath("spec").
						Child("service").Child("serviceMetadataLinks").Child("metadataIdentifier")),
				))
			}
		}

		if datasetFeed.DatasetMetadataLinks != nil && datasetFeed.SpatialDatasetIdentifierCode == nil {
			*allErrs = append(*allErrs, field.Required(
				fieldPath.Child("spatialDatasetIdentifierCode"),
				fmt.Sprintf("when %s exists", fieldPath.Child("datasetMetadataLinks").String()),
			))
		}

		if datasetFeed.SpatialDatasetIdentifierCode != nil && datasetFeed.SpatialDatasetIdentifierNamespace == nil {
			*allErrs = append(*allErrs, field.Required(
				fieldPath.Child("spatialDatasetIdentifierNamespace"),
				fmt.Sprintf("when %s exists", fieldPath.Child("spatialDatasetIdentifierCode").String()),
			))
		}

		var entryNames []string
		for in, entry := range datasetFeed.Entries {
			fieldPath = fieldPath.Child("entries").Index(in)
			if linkCount := len(entry.DownloadLinks); linkCount > 1 && entry.Content == nil {
				*allErrs = append(*allErrs, field.Required(
					fieldPath.Child("content"),
					fmt.Sprintf("when %s has 2 or more elements", fieldPath.Child("downloadlinks").String()),
				))
			}

			if slices.Contains(entryNames, entry.TechnicalName) {
				*allErrs = append(*allErrs, field.Duplicate(fieldPath.Child("technicalName"), entry.TechnicalName))
			}

			entryNames = append(entryNames, entry.TechnicalName)
		}
	}
}
