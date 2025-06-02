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
	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	"os"
	"sigs.k8s.io/yaml"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	// TODO (user): Add any additional imports if needed
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
		// TODO (user): Add logic for validating webhooks
		// Example:
		It("Should deny creation if a required field is missing", func() {
			By("simulating an invalid creation scenario")
			input, err := os.ReadFile("test_data/input/1-no-error-no-warning.yaml")
			Expect(err).NotTo(HaveOccurred())
			atom := &pdoknlv3.Atom{}
			err = yaml.Unmarshal(input, atom)
			Expect(err).NotTo(HaveOccurred())
			println(atom.Spec.Service.OwnerInfoRef)
			warnings, errors := validator.ValidateCreate(ctx, atom)
			Expect(errors).To(BeNil())
			Expect(len(warnings)).To(Equal(0))
		})

		// It("Should admit creation if all required fields are present", func() {
		//     By("simulating an invalid creation scenario")
		//     obj.SomeRequiredField = "valid_value"
		//     Expect(validator.ValidateCreate(ctx, obj)).To(BeNil())
		// })
		//
		// It("Should validate updates correctly", func() {
		//     By("simulating a valid update scenario")
		//     oldObj.SomeRequiredField = "updated_value"
		//     obj.SomeRequiredField = "updated_value"
		//     Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil())
		// })
	})

})
