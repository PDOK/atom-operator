/*
MIT License

Copyright (c) 2024 Publieke Dienstverlening op de Kaart

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package v3

import (
	"context"
	"fmt"
	"os"

	v1 "github.com/pdok/smooth-operator/api/v1"
	"github.com/pdok/smooth-operator/model"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	"sigs.k8s.io/yaml"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

var _ = Describe("Atom Webhook", func() {
	var (
		obj         *pdoknlv3.Atom
		oldObj      *pdoknlv3.Atom
		validator   AtomCustomValidator
		labelsPath  *field.Path
		servicePath *field.Path
	)

	BeforeEach(func() {
		obj = &pdoknlv3.Atom{}
		oldObj = &pdoknlv3.Atom{}
		validator = AtomCustomValidator{
			Client: k8sClient,
		}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		// TODO (user): Add any setup logic common to all tests

		labelsPath = field.NewPath("metadata").Child("labels")
		servicePath = field.NewPath("spec").Child("service")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating or updating Atom under Validating Webhook", func() {
		It("Should create atom without errors or warnings", func() {
			testCreate(validator, "minimal.yaml", nil, nil)
		})

		It("Should create atom but warn about the name containing 'atom'", func() {
			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Name += "-atom"
				},
				func(_ *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return nil, admission.Warnings{
						"pdok.nl/v3, Kind=Atom/minimal-atom: metadata.name: should not contain atom",
					}
				},
			)
		})

		It("Should deny creation if no labels are available", func() {
			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Labels = nil
				},
				func(_ *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Required(labelsPath, "can't be empty"),
					}, nil
				},
			)
		})

		It("Should deny creation if a datasetfeed metadataID is the same as the service metadataID", func() {
			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.DatasetFeeds[0].DatasetMetadataLinks.MetadataIdentifier = atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier
				},
				func(atom *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Invalid(
							servicePath.Child("datasetFeeds[0].datasetMetadataLinks.metadataIdentifier"),
							atom.Spec.Service.DatasetFeeds[0].DatasetMetadataLinks.MetadataIdentifier,
							"should not be the same as spec.service.serviceMetadataLinks.metadataIdentifier",
						),
					}, nil
				},
			)
		})

		It("Should deny creation if a datasetfeed spatialdatasetID is missing but metadatalinks is set", func() {
			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.DatasetFeeds[0].SpatialDatasetIdentifierCode = nil
				},
				func(_ *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Required(
							servicePath.Child("datasetFeeds[0].spatialDatasetIdentifierCode"),
							"when spec.service.datasetFeeds[0].datasetMetadataLinks exists",
						),
					}, nil
				},
			)
		})

		It("Should deny creation if a datasetfeed spatialdatasetID is set but spatialdatasetnamespace is not", func() {
			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.DatasetFeeds[0].SpatialDatasetIdentifierNamespace = nil
				},
				func(_ *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Required(
							servicePath.Child("datasetFeeds[0].spatialDatasetIdentifierNamespace"),
							"when spec.service.datasetFeeds[0].spatialDatasetIdentifierCode exists",
						),
					}, nil
				},
			)
		})

		It("Should deny creation if a datasetfeed entry has multiple downloadlinks but no content", func() {
			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.DatasetFeeds[0].Entries[0].DownloadLinks = append(
						atom.Spec.Service.DatasetFeeds[0].Entries[0].DownloadLinks,
						atom.Spec.Service.DatasetFeeds[0].Entries[0].DownloadLinks...,
					)
					atom.Spec.Service.DatasetFeeds[0].Entries[0].Content = nil
				},
				func(_ *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Required(
							servicePath.Child("datasetFeeds[0].entries[0].content"),
							"when spec.service.datasetFeeds[0].entries[0].downloadlinks has 2 or more elements",
						),
					}, nil
				},
			)
		})

		It("Should deny creation if datasetfeeds have duplicate technical names", func() {
			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.DatasetFeeds = append(
						atom.Spec.Service.DatasetFeeds,
						atom.Spec.Service.DatasetFeeds...,
					)
				},
				func(atom *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Duplicate(
							servicePath.Child("datasetFeeds[1].technicalName"),
							atom.Spec.Service.DatasetFeeds[1].TechnicalName,
						),
					}, nil
				},
			)
		})

		It("Should deny creation if a datasetfeed entries have duplicate technical names", func() {
			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.DatasetFeeds[0].Entries = append(
						atom.Spec.Service.DatasetFeeds[0].Entries,
						atom.Spec.Service.DatasetFeeds[0].Entries...,
					)
				},
				func(atom *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Duplicate(
							servicePath.Child("datasetFeeds[0].entries[0].entries[1].technicalName"),
							atom.Spec.Service.DatasetFeeds[0].Entries[0].TechnicalName,
						),
					}, nil
				},
			)
		})

		It("Should create atom with ingressRouteUrls that contains the service baseUrl", func() {
			testCreate(validator, "minimal.yaml", func(atom *pdoknlv3.Atom) {
				atom.Spec.IngressRouteURLs = model.IngressRouteURLs{
					{URL: atom.Spec.Service.BaseURL},
				}
			}, nil)
		})

		It("Should deny creation if ingressRouteUrls is set but does not contain the service baseUrl", func() {
			testCreate(
				validator,
				"ingress-route-urls.yaml",
				func(atom *pdoknlv3.Atom) {
					Expect(atom.Spec.IngressRouteURLs[0].URL.String()).To(Equal(atom.Spec.Service.BaseURL.String()))
					atom.Spec.IngressRouteURLs = atom.Spec.IngressRouteURLs[1:]
				},
				func(atom *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Invalid(field.NewPath("spec").Child("ingressRouteUrls"), fmt.Sprint(atom.Spec.IngressRouteURLs), "must contain baseURL: "+atom.Spec.Service.BaseURL.String()),
					}, nil
				},
			)
		})

		It("Should deny creation if spec.service.ownerReference is not found", func() {
			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.OwnerInfoRef = "random"
				},
				func(atom *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.NotFound(servicePath.Child("ownerInfoRef"), atom.Spec.Service.OwnerInfoRef),
					}, nil
				},
			)
		})

		It("Should deny creation if spec.service.ownerReference does not contain Atom info", func() {
			// Create the OwnerInfo
			ownerRef := "owner-missing-atom"
			o := v1.OwnerInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ownerRef,
					Namespace: "services",
				},
			}

			err := validator.Client.Create(context.TODO(), &o)
			Expect(err).To(Not(HaveOccurred()))

			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.OwnerInfoRef = ownerRef
				},
				func(_ *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Required(servicePath.Child("ownerInfoRef"), "spec.Atom missing in "+ownerRef),
					}, nil
				},
			)
		})

		It("Should deny creation if spec.service.ownerReference does not contain Atom info", func() {
			// Create the OwnerInfo
			ownerRef := "atom-owner-missing-templates"
			o := v1.OwnerInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ownerRef,
					Namespace: "services",
				},
				Spec: v1.OwnerInfoSpec{
					Atom: &v1.Atom{
						Author: model.Author{
							Name:  "atom",
							Email: "atom@example.com",
						},
					},
					MetadataUrls: nil,
				},
			}

			err := validator.Client.Create(context.TODO(), &o)
			Expect(err).To(Not(HaveOccurred()))

			testCreate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.OwnerInfoRef = ownerRef
				},
				func(_ *pdoknlv3.Atom) (field.ErrorList, admission.Warnings) {
					return field.ErrorList{
						field.Required(servicePath.Child("ownerInfoRef"), "spec.metadataUrls.csw missing in "+ownerRef),
						field.Required(servicePath.Child("ownerInfoRef"), "spec.metadataUrls.html missing in "+ownerRef),
						field.Required(servicePath.Child("ownerInfoRef"), "spec.metadataUrls.opensearch missing in "+ownerRef),
					}, nil
				},
			)
		})

		It("Should create and update atom without errors or warnings", func() {
			testUpdate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.Service.Title = "New service title"
				},
				nil,
			)
		})

		It("Should deny update atom with error label names cannot be added or deleted", func() {
			testUpdate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					labels := atom.GetLabels()
					labels["pdok.nl/dataset-idsssssssss"] = labels["pdok.nl/dataset-ids"]
					delete(labels, "pdok.nl/dataset-ids")
					atom.Labels = labels
				},
				func(_, _ *pdoknlv3.Atom) field.ErrorList {
					return field.ErrorList{
						field.Forbidden(labelsPath.Child("pdok.nl/dataset-idsssssssss"), "new labels cannot be added"),
					}
				},
			)
		})

		It("Should deny update atom with error label names are immutable", func() {
			testUpdate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					labels := atom.GetLabels()
					labels["pdok.nl/dataset-id"] = "dataset-changed"
					atom.Labels = labels
				},
				func(old, _ *pdoknlv3.Atom) field.ErrorList {
					return field.ErrorList{
						field.Invalid(labelsPath.Child("pdok.nl/dataset-id"), "dataset-changed", "immutable: should be: "+old.Labels["pdok.nl/dataset-id"]),
					}
				},
			)
		})

		It("Should deny update atom with error URL are immutable", func() {
			testUpdate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					// net/url.URL doesn't deepcopy...
					oldURL := atom.Spec.Service.BaseURL.String()
					newURL, _ := model.ParseURL(oldURL)
					newURL.Path += "/extra"
					atom.Spec.Service.BaseURL = model.URL{URL: newURL}
				},
				func(_, _ *pdoknlv3.Atom) field.ErrorList {
					return field.ErrorList{
						field.Forbidden(servicePath.Child("baseUrl"), "is immutable, add the old and new urls to spec.ingressRouteUrls in order to change this field"),
					}
				},
			)
		})

		It("Should deny update atom as ingressRouteURLs cannot be removed", func() {
			testUpdate(
				validator,
				"ingress-route-urls.yaml",
				func(atom *pdoknlv3.Atom) {
					atom.Spec.IngressRouteURLs = atom.Spec.IngressRouteURLs[:len(atom.Spec.IngressRouteURLs)-1]
				},
				func(_, newAtom *pdoknlv3.Atom) field.ErrorList {
					return field.ErrorList{
						field.Invalid(field.NewPath("spec").Child("ingressRouteUrls"), fmt.Sprint(newAtom.Spec.IngressRouteURLs), "urls cannot be removed, missing: {http://localhost:32788/other/path}"),
					}
				},
			)
		})

		It("Should deny update atom when the service baseUrl is changed and the old value is not added to the ingressRouteUrls", func() {
			testUpdate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					newURL, _ := model.ParseURL("http://localhost:32788/new/path")

					atom.Spec.IngressRouteURLs = model.IngressRouteURLs{{URL: model.URL{URL: newURL}}}
					atom.Spec.Service.BaseURL = model.URL{URL: newURL}
				},
				func(_, newAtom *pdoknlv3.Atom) field.ErrorList {
					return field.ErrorList{
						field.Invalid(field.NewPath("spec").Child("ingressRouteUrls"), fmt.Sprint(newAtom.Spec.IngressRouteURLs), "must contain baseURL: http://localhost:32788/owner/dataset/atom"),
					}
				},
			)
		})

		It("Should deny update atom when the service baseUrl is changed and the new value is not added to the ingressRouteUrls", func() {
			testUpdate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					oldURL := atom.Spec.Service.BaseURL
					newURL, _ := model.ParseURL("http://localhost:32788/new/path")

					atom.Spec.IngressRouteURLs = model.IngressRouteURLs{{URL: oldURL}}
					atom.Spec.Service.BaseURL = model.URL{URL: newURL}
				},
				func(_, newAtom *pdoknlv3.Atom) field.ErrorList {
					return field.ErrorList{
						field.Invalid(field.NewPath("spec").Child("ingressRouteUrls"), fmt.Sprint(newAtom.Spec.IngressRouteURLs), "must contain baseURL: http://localhost:32788/new/path"),
					}
				},
			)
		})

		It("Should create and update atom with changed service url if ingressRouteUrls is filled correctly", func() {
			testUpdate(
				validator,
				"minimal.yaml",
				func(atom *pdoknlv3.Atom) {
					oldURL := atom.Spec.Service.BaseURL
					newURL, _ := model.ParseURL("http://localhost:32788/new/path")

					atom.Spec.IngressRouteURLs = model.IngressRouteURLs{{URL: oldURL}, {URL: model.URL{URL: newURL}}}
					atom.Spec.Service.BaseURL = model.URL{URL: newURL}
				},
				nil,
			)
		})
	})
})

func testUpdate(validator AtomCustomValidator, createFile string, updateFn func(atom *pdoknlv3.Atom), errFn func(atomOld, atomNew *pdoknlv3.Atom) field.ErrorList) {
	atomOld := testCreate(validator, createFile, nil, nil)

	By("Simulating an (in)valid update scenario")
	atomNew := atomOld.DeepCopy()
	updateFn(atomNew)

	warnings, err := validator.ValidateUpdate(ctx, atomOld, atomNew)

	Expect(len(warnings)).To(Equal(0))

	if errFn == nil {
		Expect(err).To(Not(HaveOccurred()))
	} else {
		Expect(err).To(HaveOccurred())
		Expect(
			apierrors.NewInvalid(schema.GroupKind{Group: "pdok.nl", Kind: "Atom"}, atomNew.Name, errFn(atomOld, atomNew)).Error(),
		).To(Equal(err.Error()))
	}
}

func testCreate(validator AtomCustomValidator, baseFile string, mutateFn func(atom *pdoknlv3.Atom), errWarnFn func(atom *pdoknlv3.Atom) (field.ErrorList, admission.Warnings)) *pdoknlv3.Atom {
	By("simulating a (in)valid creation scenario")
	input, err := os.ReadFile("test_data/" + baseFile)
	Expect(err).NotTo(HaveOccurred())
	atom := &pdoknlv3.Atom{}
	err = yaml.Unmarshal(input, atom)
	Expect(err).NotTo(HaveOccurred())

	if mutateFn != nil {
		mutateFn(atom)
	}

	warnings, err := validator.ValidateCreate(ctx, atom)
	if errWarnFn == nil {
		Expect(len(warnings)).To(Equal(0))
		Expect(err).To(Not(HaveOccurred()))
	} else {
		expectedErrs, expectedWarns := errWarnFn(atom)
		if expectedErrs != nil {
			Expect(err).To(HaveOccurred())
			Expect(
				apierrors.NewInvalid(schema.GroupKind{Group: "pdok.nl", Kind: "Atom"}, atom.Name, expectedErrs).Error(),
			).To(Equal(err.Error()))
		}
		Expect(warnings).To(Equal(expectedWarns))
	}

	return atom
}
