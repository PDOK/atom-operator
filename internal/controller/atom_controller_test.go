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

package controller

import (
	"context"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/pdok/smooth-operator/api/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

const (
	atomResourceName      = "test-atom"
	ownerInfoResourceName = "pdok"
	namespace             = "default"
	testImageName1        = "test.test/image:test1"
	testImageName2        = "test.test/image:test2"
)

var fullAtom = pdoknlv3.Atom{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: namespace,
		Name:      atomResourceName,
		Labels: map[string]string{
			"dataset":       "test-dataset",
			"dataset-owner": "test-datasetowner",
			//"app":           "atom",
		},
	},
	Spec: pdoknlv3.AtomSpec{
		Service: pdoknlv3.Service{
			BaseURL:      "https://my.test-resource.test/atom",
			Lang:         "test lang",
			Stylesheet:   "test stylesheet",
			Title:        "test title",
			Subtitle:     "test subtitle",
			OwnerInfoRef: ownerInfoResourceName,
			ServiceMetadataLinks: pdoknlv3.MetadataLink{
				MetadataIdentifier: "00000000-0000-0000-0000-000000000000",
				Templates:          []string{"csw", "opensearch", "html"},
			},
			Rights: "test rights",
		},
		DatasetFeeds: []pdoknlv3.DatasetFeed{},
		PodSpecPatch: &corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name: "init-container",
					VolumeMounts: []corev1.VolumeMount{
						{Name: "data", MountPath: srvDir + "/data"},
						{Name: "config", MountPath: srvDir + "/config"},
					},
					Image: testImageName1,
				},
			},
			Containers: []corev1.Container{
				{
					Name: "atom-service",
					VolumeMounts: []corev1.VolumeMount{
						{Name: "socket", MountPath: "/tmp", ReadOnly: false},
						{Name: "data", MountPath: "var/www"},
					},
					Image: testImageName2,
				},
			},
			Volumes: []corev1.Volume{},
		},
	},
}

var _ = Describe("Atom Controller", func() {
	Context("When reconciling a resource", func() {

		ctx := context.Background()

		typeNamespacedNameAtom := types.NamespacedName{
			Name:      atomResourceName,
			Namespace: namespace,
		}
		atom := &pdoknlv3.Atom{}

		typeNamespacedNameOwnerInfo := types.NamespacedName{
			Namespace: namespace,
			Name:      ownerInfoResourceName,
		}
		ownerInfo := &v1.OwnerInfo{}

		BeforeEach(func() {

			By("creating the custom resource for the Kind Atom")
			err := k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			if err != nil && errors.IsNotFound(err) {
				resource := fullAtom.DeepCopy()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("creating the custom resource for the Kind OwnerInfo")
			err = k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfo)
			if err != nil && errors.IsNotFound(err) { //

				resource := &v1.OwnerInfo{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      ownerInfoResourceName,
					},
					// TODO(user): Specify other spec details if needed.
					// Author
					// CSW template
					// Opensearch template
					// HTML template
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			atomResource := &pdoknlv3.Atom{}
			err := k8sClient.Get(ctx, typeNamespacedNameAtom, atomResource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Atom")
			Expect(k8sClient.Delete(ctx, atomResource)).To(Succeed())

			ownerInfoResource := &v1.OwnerInfo{}
			err = k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfoResource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance OwnerInfo")
			Expect(k8sClient.Delete(ctx, ownerInfoResource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedNameAtom,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}
