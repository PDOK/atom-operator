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
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/google/go-cmp/cmp"
	"github.com/pdok/atom-generator/feeds"
	"sigs.k8s.io/yaml"

	"testing"
	"time"

	policyv1 "k8s.io/api/policy/v1"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"
	smoothoperatorvalidation "github.com/pdok/smooth-operator/pkg/validation"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

const (
	namespace      = "default"
	testImageName1 = "test.test/image:test1"
	testImageName2 = "test.test/image:test2"
)

var _ = Describe("Testing Atom Controller", func() {

	Context("Testing Mutate functions for Minimal Atom", func() {

		pdoknlv3.SetBlobEndpoint("http://localazurite.blob.azurite")

		var reconciler AtomReconciler

		testPath := "test_data/minimal-atom"
		outputPath := testPath + "/expected-output/"

		atom := pdoknlv3.Atom{}
		owner := smoothoperatorv1.OwnerInfo{}

		BeforeEach(func() {
			reconciler = AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}
		})

		It("Should parse the input files correctly", func() {

			data, err := os.ReadFile(testPath + "/input/atom.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = yaml.UnmarshalStrict(data, &atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Name).Should(Equal("minimal"))

			data, err = os.ReadFile(testPath + "/input/ownerinfo.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = yaml.UnmarshalStrict(data, &owner)
			Expect(err).NotTo(HaveOccurred())
			Expect(owner.Name).Should(Equal("owner"))
		})

		It("Should generate a correct Configmap", func() {

			result := getBareConfigMap(&atom)
			err := reconciler.mutateAtomGeneratorConfigMap(&atom, &owner, result)
			Expect(err).NotTo(HaveOccurred())

			var expected corev1.ConfigMap
			data, err := os.ReadFile(outputPath + "configmap.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = yaml.UnmarshalStrict(data, &expected)
			Expect(err).NotTo(HaveOccurred())

			diff := cmp.Diff(expected, *result)
			if diff != "" {

				var expectedValues, gottenValues feeds.Feeds
				err = yaml.UnmarshalStrict([]byte(expected.Data["values.yaml"]), &expectedValues)
				Expect(err).NotTo(HaveOccurred())
				err = yaml.UnmarshalStrict([]byte(result.Data["values.yaml"]), &gottenValues)
				Expect(err).NotTo(HaveOccurred())

				valuesDiff := cmp.Diff(expectedValues, gottenValues)
				if valuesDiff != "" {
					Fail(valuesDiff)
				}

				Fail(diff)
			}
		})

		It("Should generate a correct Deployment", func() {
			testMutate(getBareDeployment(&atom), outputPath+"deployment.yaml", func(d *appsv1.Deployment) error {
				return reconciler.mutateDeployment(&atom, d, "minimal-atom-generator")
			})
		})

		It("Should generate a correct Service", func() {
			testMutate(getBareService(&atom), outputPath+"service.yaml", func(s *corev1.Service) error {
				return reconciler.mutateService(&atom, s)
			})
		})

		It("Should generate a correct Prefix Strip Middleware", func() {
			testMutate(getBareStripPrefixMiddleware(&atom), outputPath+"middleware-prefixstrip.yaml", func(m *traefikiov1alpha1.Middleware) error {
				return reconciler.mutateStripPrefixMiddleware(&atom, m)
			})
		})

		It("Should generate a correct Headers Middleware", func() {
			testMutate(getBareHeadersMiddleware(&atom), outputPath+"middleware-headers.yaml", func(m *traefikiov1alpha1.Middleware) error {
				return reconciler.mutateHeadersMiddleware(&atom, m)
			})
		})

		It("Should generate a correct Download Middleware", func() {
			testMutate(getBareDownloadLinkMiddleware(&atom, 0), outputPath+"middleware-downloads.yaml", func(m *traefikiov1alpha1.Middleware) error {
				return reconciler.mutateDownloadLinkMiddleware(&atom, &atom.Spec.Service.DatasetFeeds[0].Entries[0].DownloadLinks[0], m)
			})
		})

		It("Should generate a correct IngressRoute", func() {
			testMutate(getBareIngressRoute(&atom), outputPath+"ingressroute.yaml", func(i *traefikiov1alpha1.IngressRoute) error {
				return reconciler.mutateIngressRoute(&atom, i)
			})
		})

		It("Should generate a correct PodDisruptionBudget", func() {
			testMutate(getBarePodDisruptionBudget(&atom), outputPath+"poddisruptionbudget.yaml", func(p *policyv1.PodDisruptionBudget) error {
				return reconciler.mutatePodDisruptionBudget(&atom, p)
			})
		})
	})

	Context("When reconciling a resource", func() {

		ctx := context.Background()

		testPath := "test_data/minimal-atom/input/"

		testAtom := pdoknlv3.Atom{}
		clusterAtom := &pdoknlv3.Atom{}

		objectKeyAtom := types.NamespacedName{}

		testOwner := smoothoperatorv1.OwnerInfo{}
		clusterOwner := &smoothoperatorv1.OwnerInfo{}

		objectKeyOwner := types.NamespacedName{}

		var expectedResources []struct {
			obj client.Object
			key types.NamespacedName
		}

		It("Should create a Atom and OwnerInfo resource on the cluster", func() {

			By("Creating a new resource for the Kind Atom")
			data, err := os.ReadFile(testPath + "atom.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = yaml.UnmarshalStrict(data, &testAtom)
			Expect(err).NotTo(HaveOccurred())
			Expect(testAtom.Name).Should(Equal("minimal"))

			objectKeyAtom = types.NamespacedName{
				Namespace: testAtom.GetNamespace(),
				Name:      testAtom.GetName(),
			}

			data, err = os.ReadFile(testPath + "ownerinfo.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = yaml.UnmarshalStrict(data, &testOwner)
			Expect(err).NotTo(HaveOccurred())
			Expect(testOwner.Name).Should(Equal("owner"))

			objectKeyOwner = types.NamespacedName{
				Namespace: testOwner.GetNamespace(),
				Name:      testOwner.GetName(),
			}

			err = k8sClient.Get(ctx, objectKeyAtom, clusterAtom)
			if err != nil && apierrors.IsNotFound(err) {
				resource := testAtom.DeepCopy()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, objectKeyAtom, clusterAtom)).To(Succeed())
			}

			By("Creating a new resource for the Kind OwnerInfo")
			err = k8sClient.Get(ctx, objectKeyOwner, clusterOwner)
			if err != nil && apierrors.IsNotFound(err) {
				resource := testOwner.DeepCopy()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, objectKeyOwner, clusterOwner)).To(Succeed())
			}
		})

		It("Should generate all expected resources after a Reconcile", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom and checking the deployment manifest")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyAtom})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create all expected resources", func() {
			configMapName, err := getAtomConfigMapNameFromClient(ctx, clusterAtom)
			Expect(err).NotTo(HaveOccurred())
			expectedResources = getExpectedBareObjectsForAtom(clusterAtom, configMapName)

			for _, expectedResource := range expectedResources {
				Eventually(func() bool {
					err := k8sClient.Get(ctx, expectedResource.key, expectedResource.obj)
					return Expect(err).NotTo(HaveOccurred())
				}, "10s", "1s").Should(BeTrue())
			}
		})

		It("Should successfully reconcile after a change in an owned resource", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Getting the original Deployment")
			deployment := getBareDeployment(clusterAtom)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())
			originalMinReadySeconds := deployment.Spec.MinReadySeconds
			expectedMinReadySeconds := 99
			Expect(originalMinReadySeconds).Should(Not(Equal(expectedMinReadySeconds)))

			By("Altering the Deployment")
			err := k8sClient.Patch(ctx, deployment, client.RawPatch(types.MergePatchType, []byte(
				fmt.Sprintf(`{"spec": {"minReadySeconds": %d}}`, expectedMinReadySeconds))))
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was altered")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(deployment.Spec.MinReadySeconds).To(BeEquivalentTo(expectedMinReadySeconds))
			}, "10s", "1s").Should(BeTrue())

			By("Reconciling the Atom again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyAtom})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was restored")
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(deployment.Spec.MinReadySeconds).To(BeEquivalentTo(originalMinReadySeconds))
			}, "10s", "1s").Should(BeTrue())
		})

		It("Should cleanup the cluster", func() {
			err := k8sClient.Get(ctx, objectKeyAtom, clusterAtom)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Atom")
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, clusterAtom))).To(Succeed())

			err = k8sClient.Get(ctx, objectKeyOwner, clusterOwner)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance OwnerInfo")
			Expect(k8sClient.Delete(ctx, clusterOwner)).To(Succeed())

			//the testEnv does not do garbage collection (https://book.kubebuilder.io/reference/envtest#testing-considerations)
			By("Cleaning Owned Resources")
			for _, d := range expectedResources {
				err := k8sClient.Get(ctx, d.key, d.obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Delete(ctx, d.obj)).To(Succeed())
			}
		})
	})

	Context("When manually validating an incoming CRD", func() {
		It("Should not error", func() {
			err := smoothoperatorvalidation.LoadSchemasForCRD(cfg, "default", "atoms.pdok.nl")
			Expect(err).NotTo(HaveOccurred())

			yamlInput := readTestFile("crd/v3_atom.yaml")

			err = smoothoperatorvalidation.ValidateSchema(yamlInput)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

// TODO move to smoothOperator?
func testMutate[T any](result *T, expectedFile string, mutate func(*T) error) {
	err := mutate(result)
	Expect(err).NotTo(HaveOccurred())

	var expected T
	data, err := os.ReadFile(expectedFile)
	Expect(err).NotTo(HaveOccurred())
	err = yaml.UnmarshalStrict(data, &expected)
	Expect(err).NotTo(HaveOccurred())

	diff := cmp.Diff(expected, *result)
	if diff != "" {
		Fail(diff)
	}
}

func getAtomConfigMapNameFromClient(ctx context.Context, atom *pdoknlv3.Atom) (string, error) {
	deployment := &appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: atom.GetNamespace(), Name: getBareDeployment(atom).GetName()}, deployment)
	if err != nil {
		return "", err
	}
	return getAtomConfigMapNameFromDeployment(deployment)
}

func getAtomConfigMapNameFromDeployment(deployment *appsv1.Deployment) (string, error) {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == "config" && volume.ConfigMap != nil {
			return volume.ConfigMap.Name, nil
		}
	}
	return "", errors.New("AtomOperator deployment configmap not found")
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func getExpectedBareObjectsForAtom(atom *pdoknlv3.Atom, configMapName string) []struct {
	obj client.Object
	key types.NamespacedName
} {
	structs := []struct {
		obj client.Object
		key types.NamespacedName
	}{
		{obj: &appsv1.Deployment{}, key: types.NamespacedName{Namespace: namespace, Name: getBareDeployment(atom).GetName()}},
		{obj: &corev1.ConfigMap{}, key: types.NamespacedName{Namespace: namespace, Name: configMapName}},
		{obj: &traefikiov1alpha1.Middleware{}, key: types.NamespacedName{Namespace: namespace, Name: getBareStripPrefixMiddleware(atom).GetName()}},
		{obj: &traefikiov1alpha1.Middleware{}, key: types.NamespacedName{Namespace: namespace, Name: getBareHeadersMiddleware(atom).GetName()}},
		{obj: &corev1.Service{}, key: types.NamespacedName{Namespace: namespace, Name: getBareService(atom).GetName()}},
		{obj: &traefikiov1alpha1.IngressRoute{}, key: types.NamespacedName{Namespace: namespace, Name: getBareIngressRoute(atom).GetName()}},
		{obj: &policyv1.PodDisruptionBudget{}, key: types.NamespacedName{Namespace: namespace, Name: getBarePodDisruptionBudget(atom).GetName()}},
	}
	for index := range atom.GetDownloadLinks() {
		extraStruct := struct {
			obj client.Object
			key types.NamespacedName
		}{obj: &traefikiov1alpha1.Middleware{}, key: types.NamespacedName{Namespace: namespace, Name: getBareDownloadLinkMiddleware(atom, index).GetName()}}

		structs = append(structs, extraStruct)
	}

	return structs
}

func Test_getGeneratorConfig(t *testing.T) {
	type args struct {
		atom      *pdoknlv3.Atom
		ownerInfo *smoothoperatorv1.OwnerInfo
	}
	tests := []struct {
		name       string
		args       args
		wantConfig string
		wantErr    bool
	}{
		// TODO: Add test cases.
		{
			name: "error_empty_scenario_01",
			args: args{
				atom:      &pdoknlv3.Atom{},
				ownerInfo: &smoothoperatorv1.OwnerInfo{},
			},
			wantConfig: "",
			wantErr:    true,
		},
		{
			name: "succesfull_scenario_02",
			args: args{
				atom: &pdoknlv3.Atom{
					Spec: pdoknlv3.AtomSpec{
						Lifecycle: &smoothoperatormodel.Lifecycle{},
						Service: pdoknlv3.Service{
							BaseURL:    "/",
							Stylesheet: smoothutil.Pointer("/atom/style/style.xsl"),
							Lang:       "nl",
							ServiceMetadataLinks: &pdoknlv3.MetadataLink{
								MetadataIdentifier: "7c5bbc80-d6f1-48d7-ba75-xxxxxxxxxxxx",
								Templates:          []string{"csw", "opensearch", "html"},
							},
							DatasetFeeds: []pdoknlv3.DatasetFeed{
								{
									TechnicalName: "brocpt",
									Title:         "BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM",
									Subtitle:      "BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM",
									//Links:         []pdoknlv3.Link{},
									DatasetMetadataLinks: &pdoknlv3.MetadataLink{
										MetadataIdentifier: "d893c05b-907e-47f2-9cbd-ceb08e68732c",
										Templates:          []string{"csw", "html"},
									},
									Author: smoothoperatormodel.Author{
										Name:  "owner",
										Email: "info@test.com",
									},
									SpatialDatasetIdentifierCode:      smoothutil.Pointer("d893c05b-907e-47f2-9cbd-ceb08e68732c"),
									SpatialDatasetIdentifierNamespace: smoothutil.Pointer("http://www.pdok.nl"),
									Entries: []pdoknlv3.Entry{
										{
											TechnicalName: "bro_geotechnisch_sondeeronderzoek_cpt_inspire_geharmoniseerd_geologie",
											Title:         smoothutil.Pointer("BRO - Geotechnisch sondeeronderzoek (CPT) INSPIRE geharmoniseerd - Geologie"),
											Content:       smoothutil.Pointer("Gegevens van geotechnisch sondeeronderzoek (kenset) zoals opgeslagen in de Basis Registratie Ondergrond (BRO)."),
											DownloadLinks: []pdoknlv3.DownloadLink{
												{
													Data: "http://localazurite.blob.azurite/bucket/key1/dataset-1-file",
												},
											},
											Polygon: getTestPolygon(),
											Updated: metav1.Time{Time: getUpdatedDate()},
											SRS: pdoknlv3.SRS{
												Name: "Amersfoort / RD New",
												URI:  "https://www.opengis.net/def/crs/EPSG/0/28992",
											},
										},
									},
								},
							},
						},
					},
				},
				ownerInfo: &smoothoperatorv1.OwnerInfo{
					Spec: smoothoperatorv1.OwnerInfoSpec{
						MetadataUrls: &smoothoperatorv1.MetadataUrls{
							CSW: &smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
							},
							OpenSearch: &smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/opensearch/dut/{{identifier}}/OpenSearchDescription.xml",
							},
							HTML: &smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/catalog.search#/metadata/{{identifier}}",
							},
						},
						Atom: &smoothoperatorv1.Atom{
							Author: smoothoperatormodel.Author{
								Name:  "owner",
								Email: "info@test.com",
							},
						},
					},
				},
			},
			wantConfig: readTestFile("generator_config/scenario-2.yaml"),
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConfig, err := getGeneratorConfig(tt.args.atom, tt.args.ownerInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("getGeneratorConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			require.YAMLEqf(t, tt.wantConfig, gotConfig, "getGeneratorConfig() gotConfig = %v, want %v", gotConfig, tt.wantConfig)
		})
	}
}

func readTestFile(fileName string) string {
	dat, _ := os.ReadFile("test_data/" + fileName)

	return string(dat)
}

func getTestPolygon() pdoknlv3.Polygon {
	return pdoknlv3.Polygon{
		BBox: smoothoperatormodel.BBox{
			MinX: "1",
			MinY: "1",
			MaxX: "2",
			MaxY: "2",
		},
	}
}

func getUpdatedDate() time.Time {
	return metav1.Date(2025, time.March, 5, 5, 5, 5, 0, time.UTC).UTC()
}
