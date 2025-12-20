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

	ValidateAtom(atom, &warnings, &allErrs)
	ValidateOwnerInfo(c, atom, &allErrs)

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

	if atom.Spec.IngressRouteURLs == nil {
		smoothoperatorvalidation.CheckURLImmutability(
			atomOld.Spec.Service.BaseURL,
			atom.Spec.Service.BaseURL,
			&allErrs,
			field.NewPath("spec").Child("service").Child("baseUrl"),
		)
	} else if atom.Spec.Service.BaseURL.String() != atomOld.Spec.Service.BaseURL.String() {
		err := smoothoperatorvalidation.ValidateIngressRouteURLsContainsBaseURL(atom.Spec.IngressRouteURLs, atomOld.Spec.Service.BaseURL, nil)
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}

	smoothoperatorvalidation.ValidateIngressRouteURLsNotRemoved(atomOld.Spec.IngressRouteURLs, atom.Spec.IngressRouteURLs, &allErrs, nil)

	ValidateAtom(atom, &warnings, &allErrs)
	ValidateOwnerInfo(c, atom, &allErrs)

	if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, apierrors.NewInvalid(
		schema.GroupKind{Group: "pdok.nl", Kind: "Atom"},
		atom.Name, allErrs)
}

func ValidateOwnerInfo(c client.Client, atom *Atom, allErrs *field.ErrorList) {
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
	} else {
		validateMetadataTemplates(atom, ownerInfo, allErrs)
	}
}

func validateMetadataTemplates(atom *Atom, ownerInfo *smoothoperatorv1.OwnerInfo, allErrs *field.ErrorList) {
	var metadataTemplates []string
	if atom.Spec.Service.ServiceMetadataLinks != nil {
		metadataTemplates = atom.Spec.Service.ServiceMetadataLinks.Templates
	}
	for _, feed := range atom.Spec.Service.DatasetFeeds {
		if feed.DatasetMetadataLinks != nil {
			metadataTemplates = append(metadataTemplates, feed.DatasetMetadataLinks.Templates...)
		}
	}

	if len(metadataTemplates) > 0 {
		fieldPath := field.NewPath("spec").Child("service").Child("ownerInfoRef")
		if slices.Contains(metadataTemplates, "csw") && (ownerInfo.Spec.MetadataUrls == nil || ownerInfo.Spec.MetadataUrls.CSW == nil) {
			*allErrs = append(*allErrs, field.Required(fieldPath, "spec.metadataUrls.csw missing in "+ownerInfo.Name))
		}
		if slices.Contains(metadataTemplates, "html") && (ownerInfo.Spec.MetadataUrls == nil || ownerInfo.Spec.MetadataUrls.HTML == nil) {
			*allErrs = append(*allErrs, field.Required(fieldPath, "spec.metadataUrls.html missing in "+ownerInfo.Name))
		}
		if slices.Contains(metadataTemplates, "opensearch") && (ownerInfo.Spec.MetadataUrls == nil || ownerInfo.Spec.MetadataUrls.OpenSearch == nil) {
			*allErrs = append(*allErrs, field.Required(fieldPath, "spec.metadataUrls.opensearch missing in "+ownerInfo.Name))
		}
	}
}

func ValidateAtom(atom *Atom, warnings *[]string, allErrs *field.ErrorList) {
	var fieldPath *field.Path
	if strings.Contains(atom.GetName(), "atom") {
		fieldPath = field.NewPath("metadata").Child("name")
		smoothoperatorvalidation.AddWarning(warnings, *fieldPath, "should not contain atom", atom.GroupVersionKind(), atom.GetName())
	}

	validateDatasetFeeds(atom, allErrs)

	err := smoothoperatorvalidation.ValidateIngressRouteURLsContainsBaseURL(atom.Spec.IngressRouteURLs, atom.Spec.Service.BaseURL, nil)
	if err != nil {
		*allErrs = append(*allErrs, err)
	}
}

func validateDatasetFeeds(atom *Atom, allErrs *field.ErrorList) {
	var feedNames []string
	for i, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		fieldPath := field.NewPath("spec").Child("service").Child("datasetFeeds").Index(i)

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
