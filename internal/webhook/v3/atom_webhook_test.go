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
	"fmt"
	"os"
	"strings"

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
			By("simulating a valid creation scenario")
			input, err := os.ReadFile("test_data/input/1-create-no-error-no-warning.yaml")
			Expect(err).NotTo(HaveOccurred())
			atom := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atom)
			Expect(err).NotTo(HaveOccurred())
			warnings, errors := validator.ValidateCreate(ctx, atom)
			Expect(errors).To(BeNil())
			Expect(len(warnings)).To(Equal(0))
		})

		It("Should deny creation if no labels are available", func() {
			By("simulating an invalid creation scenario")
			input, err := os.ReadFile("test_data/input/2-create-error-no-lables.yaml")
			Expect(err).NotTo(HaveOccurred())
			atom := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atom)
			Expect(err).NotTo(HaveOccurred())
			warnings, errorsCreate := validator.ValidateCreate(ctx, atom)

			expectedError := errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: metadata.labels: Required value: can't be empty")
			Expect(len(warnings)).To(Equal(0))
			Expect(expectedError.Error()).To(Equal(errorsCreate.Error()))
		})

		It("Should create and update atom without errors or warnings", func() {
			By("simulating a valid creation scenario")
			input, err := os.ReadFile("test_data/input/1-create-no-error-no-warning.yaml")
			Expect(err).NotTo(HaveOccurred())
			atomOld := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atomOld)
			Expect(err).NotTo(HaveOccurred())
			warnings, errors := validator.ValidateCreate(ctx, atomOld)
			Expect(errors).To(BeNil())
			Expect(len(warnings)).To(Equal(0))

			By("simulating a valid update scenario")
			input, err = os.ReadFile("test_data/input/3-update-no-error-no-warning.yaml")
			Expect(err).NotTo(HaveOccurred())
			atomNew := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atomNew)
			Expect(err).NotTo(HaveOccurred())
			warnings, errors = validator.ValidateUpdate(ctx, atomOld, atomNew)
			Expect(errors).To(BeNil())
			Expect(len(warnings)).To(Equal(0))
		})

		It("Should deny update atom with error label names cannot be added or deleted", func() {
			By("simulating a valid creation scenario")
			input, err := os.ReadFile("test_data/input/1-create-no-error-no-warning.yaml")
			Expect(err).NotTo(HaveOccurred())
			atomOld := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atomOld)
			Expect(err).NotTo(HaveOccurred())
			warningsCreate, errorsCreate := validator.ValidateCreate(ctx, atomOld)
			Expect(errorsCreate).To(BeNil())
			Expect(len(warningsCreate)).To(Equal(0))

			By("simulating an invalid update scenario. error label names cannot be added or deleted")
			input, err = os.ReadFile("test_data/input/4-update-error-add-or-delete-labels.yaml")
			Expect(err).NotTo(HaveOccurred())
			atomNew := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atomNew)
			Expect(err).NotTo(HaveOccurred())
			warningsUpdate, errorsUpdate := validator.ValidateUpdate(ctx, atomOld, atomNew)

			expectedError := errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: [metadata.labels.pdok.nl/dataset-id: Required value: labels cannot be removed, metadata.labels.pdok.nl/dataset-idsssssssss: Forbidden: new labels cannot be added]")
			Expect(len(warningsUpdate)).To(Equal(0))
			Expect(expectedError.Error()).To(Equal(errorsUpdate.Error()))
		})

		It("Should deny update atom with error label names are immutable", func() {
			By("simulating a valid creation scenario")
			input, err := os.ReadFile("test_data/input/1-create-no-error-no-warning.yaml")
			Expect(err).NotTo(HaveOccurred())
			atomOld := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atomOld)
			Expect(err).NotTo(HaveOccurred())
			warningsCreate, errorsCreate := validator.ValidateCreate(ctx, atomOld)
			Expect(errorsCreate).To(BeNil())
			Expect(len(warningsCreate)).To(Equal(0))

			By("simulating an invalid update scenario. Lablels are immutable")
			input, err = os.ReadFile("test_data/input/5-update-error-labels-immutable.yaml")
			Expect(err).NotTo(HaveOccurred())
			atomNew := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atomNew)
			Expect(err).NotTo(HaveOccurred())
			warningsUpdate, errorsUpdate := validator.ValidateUpdate(ctx, atomOld, atomNew)

			fmt.Printf("actual-error test 5 atom-webhook is: \n%v\n", errorsUpdate.Error())
			expectedError := errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: metadata.labels.pdok.nl/dataset-id: Invalid value: \"wetlands-changed\": immutable: should be wetlands")
			Expect(strings.ReplaceAll(expectedError.Error(), ":", "")).To(Equal(strings.ReplaceAll(errorsUpdate.Error(), ":", "")))
			Expect(len(warningsUpdate)).To(Equal(0))
		})

		It("Should deny update atom with error URL are immutable", func() {
			By("simulating a valid creation scenario")
			input, err := os.ReadFile("test_data/input/1-create-no-error-no-warning.yaml")
			Expect(err).NotTo(HaveOccurred())
			atomOld := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atomOld)
			Expect(err).NotTo(HaveOccurred())
			warnings, errorsCreate := validator.ValidateCreate(ctx, atomOld)
			Expect(errorsCreate).To(BeNil())
			Expect(len(warnings)).To(Equal(0))

			By("simulating an invalid update scenario. URL is immutable")
			input, err = os.ReadFile("test_data/input/6-update-error-url-immutable.yaml")
			Expect(err).NotTo(HaveOccurred())
			atomNew := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atomNew)
			Expect(err).NotTo(HaveOccurred())
			warnings, errorsUpdate := validator.ValidateUpdate(ctx, atomOld, atomNew)

			expectedError := errors.New("Atom.pdok.nl \"asis-readonly-prod\" is invalid: spec.service.baseUrl: Forbidden: is immutable")
			Expect(len(warnings)).To(Equal(0))
			Expect(expectedError.Error()).To(Equal(errorsUpdate.Error()))
		})
	})
})
