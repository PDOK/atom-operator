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
	"errors"
	"os"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	"sigs.k8s.io/yaml"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

var _ = Describe("Atom Webhook", func() {
	var (
		obj       *pdoknlv3.Atom
		oldObj    *pdoknlv3.Atom
		validator AtomCustomValidator
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

	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating or updating Atom under Validating Webhook", func() {
		It("Should create atom without errors or warnings", func() {
			testCreate(validator, "valid/minimal.yaml", nil)
		})

		It("Should deny creation if no labels are available", func() {
			testCreate(validator, "invalid/no-labels.yaml", errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: metadata.labels: Required value: can't be empty"))
		})

		It("Should create atom with ingressRouteUrls that contains the service baseUrl", func() {
			testCreate(validator, "valid/ingress-route-urls.yaml", nil)
		})

		It("Should deny creation if ingressRouteUrls is set but does not contain the service baseUrl", func() {
			testCreate(
				validator,
				"invalid/ingress-route-urls-missing-baseurl.yaml",
				errors.New("Atom.pdok.nl \"ingress-route-urls\" is invalid: spec.ingressRouteUrls: Invalid value: \"[{http://test.com/path}]\": must contain baseURL: http://localhost:32788/rvo/wetlands/atom"),
			)
		})

		It("Should create and update atom without errors or warnings", func() {
			testUpdate(validator, "valid/minimal.yaml", "valid/minimal-service-title-changed.yaml", nil)
		})

		It("Should deny update atom with error label names cannot be added or deleted", func() {
			testUpdate(
				validator,
				"valid/minimal.yaml",
				"invalid/minimal-immutable-labels-key-change.yaml",
				errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: [metadata.labels.pdok.nl/dataset-id: Required value: labels cannot be removed, metadata.labels.pdok.nl/dataset-idsssssssss: Forbidden: new labels cannot be added]"),
			)
		})

		It("Should deny update atom with error label names are immutable", func() {
			testUpdate(
				validator,
				"valid/minimal.yaml",
				"invalid/minimal-immutable-labels-value-change.yaml",
				errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: metadata.labels.pdok.nl/dataset-id: Invalid value: \"wetlands-changed\": immutable: should be: wetlands"),
			)
		})

		It("Should deny update atom with error URL are immutable", func() {
			testUpdate(
				validator,
				"valid/minimal.yaml",
				"invalid/minimal-immutable-url.yaml", errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: spec.service.baseUrl: Forbidden: is immutable"),
			)
		})

		It("Should deny update atom as ingressRouteURLs cannot be removed", func() {
			testUpdate(
				validator,
				"valid/ingress-route-urls.yaml",
				"invalid/ingress-route-urls-removed-url.yaml",
				errors.New("Atom.pdok.nl \"ingress-route-urls\" is invalid: spec.ingressRouteUrls: Invalid value: \"[{http://localhost:32788/rvo/wetlands/atom}]\": urls cannot be removed, missing: {http://localhost:32788/other/path}"),
			)
		})

		It("Should deny update atom when the service baseUrl is changed and the old value is not added to the ingressRouteUrls", func() {
			testUpdate(
				validator,
				"valid/minimal.yaml",
				"invalid/minimal-service-url-changed-ingress-route-urls-missing-old.yaml",
				errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: spec.ingressRouteUrls: Invalid value: \"[{http://localhost:32788/new/path}]\": must contain baseURL: http://localhost:32788/rvo/wetlands/atom"),
			)
		})

		It("Should deny update atom when the service baseUrl is changed and the new value is not added to the ingressRouteUrls", func() {
			testUpdate(
				validator,
				"valid/minimal.yaml",
				"invalid/minimal-service-url-changed-ingress-route-urls-missing-new.yaml",
				errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: spec.ingressRouteUrls: Invalid value: \"[{http://localhost:32788/rvo/wetlands/atom}]\": must contain baseURL: http://localhost:32788/new/path"),
			)
		})

		It("Should create and update atom with changed service url if ingressRouteUrls is filled correctly", func() {
			testUpdate(validator, "valid/minimal.yaml", "valid/minimal-service-url-changed.yaml", nil)
		})
	})
})

func testUpdate(validator AtomCustomValidator, createFile, updateFile string, expectedError error) {
	atomOld := testCreate(validator, createFile, nil)

	By("Simulating an (in)valid update scenario")
	input, err := os.ReadFile("test_data/updates/" + updateFile)
	Expect(err).NotTo(HaveOccurred())
	atomNew := &pdoknlv3.Atom{}
	err = yaml.Unmarshal(input, atomNew)
	Expect(err).NotTo(HaveOccurred())
	Expect(atomOld.GetName()).To(Equal(atomNew.GetName()))
	warnings, errorsUpdate := validator.ValidateUpdate(ctx, atomOld, atomNew)

	Expect(len(warnings)).To(Equal(0))

	if expectedError == nil {
		Expect(errorsUpdate).To(Not(HaveOccurred()))
	} else {
		Expect(errorsUpdate).To(HaveOccurred())
		Expect(expectedError.Error()).To(Equal(errorsUpdate.Error()))
	}
}

func testCreate(validator AtomCustomValidator, createFile string, expectedError error) *pdoknlv3.Atom {
	By("simulating a (in)valid creation scenario")
	input, err := os.ReadFile("test_data/creates/" + createFile)
	Expect(err).NotTo(HaveOccurred())
	atom := &pdoknlv3.Atom{}
	err = yaml.Unmarshal(input, atom)
	Expect(err).NotTo(HaveOccurred())
	warnings, err := validator.ValidateCreate(ctx, atom)
	Expect(len(warnings)).To(Equal(0))

	if expectedError == nil {
		Expect(err).To(Not(HaveOccurred()))
	} else {
		Expect(expectedError.Error()).To(Equal(err.Error()))
	}

	return atom
}
