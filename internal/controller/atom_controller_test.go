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
	"log"
	"sync/atomic"
	"testing"
	"time"
	"unicode"

	"github.com/pkg/errors"
	traefikiov1alpha1 "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	policyv1 "k8s.io/api/policy/v1"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

const (
	atomResourceName      = "test-atom"
	ownerInfoResourceName = "pdok"
	namespace             = "default"
	testImageName1        = "test.test/image:test1"
	testImageName2        = "test.test/image:test2"
)

var updated = metav1.NewTime(time.Now())
var fullAtom = pdoknlv3.Atom{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: namespace,
		Name:      atomResourceName,
		Labels: map[string]string{
			"dataset":       "test-dataset",
			"dataset-owner": "test-datasetowner",
		},
	},
	Spec: pdoknlv3.AtomSpec{
		Lifecycle: smoothoperatormodel.Lifecycle{
			TTLInDays: int32Ptr(999),
		},
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
		DatasetFeeds: []pdoknlv3.DatasetFeed{
			{
				TechnicalName: "test-technical-name",
				Title:         "test-title",
				Subtitle:      "test-subtitle",
				DatasetMetadataLinks: pdoknlv3.MetadataLink{
					MetadataIdentifier: "11111111-1111-1111-1111-111111111111",
					Templates:          []string{"csw", "html"},
				},
				SpatialDatasetIdentifierCode:      "22222222-2222-2222-2222-222222222222",
				SpatialDatasetIdentifierNamespace: "http://www.pdok.nl",
				Entries: []pdoknlv3.Entry{
					{
						TechnicalName: "test-technical-name",
						DownloadLinks: []pdoknlv3.DownloadLink{
							{
								Data: "http://localazurite.blob.azurite/bucket/key1/dataset-1-file",
								BBox: &smoothoperatormodel.BBox{
									MinX: "482.06",
									MinY: "284182.97",
									MaxX: "306602.42",
									MaxY: "637049.52",
								},
							},
						},
						Updated: &updated,
						Polygon: &pdoknlv3.Polygon{
							BBox: smoothoperatormodel.BBox{
								MinX: "482.06",
								MinY: "284182.97",
								MaxX: "306602.42",
								MaxY: "637049.52",
							},
						},
						SRS: &pdoknlv3.SRS{
							URI:  "https://www.opengis.net/def/crs/EPSG/0/28992",
							Name: "Amersfoort / RD New",
						},
					},
				},
			},
		},
		PodSpecPatch: &corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name: "init-atom",
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
		ownerInfo := &smoothoperatorv1.OwnerInfo{}

		BeforeEach(func() {

			By("creating the custom resource for the Kind Atom")
			err := k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			if err != nil && k8serrors.IsNotFound(err) {
				resource := fullAtom.DeepCopy()
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedNameAtom, atom)).To(Succeed())
			}

			By("creating the custom resource for the Kind OwnerInfo")
			err = k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfo)
			if err != nil && k8serrors.IsNotFound(err) {

				resource := &smoothoperatorv1.OwnerInfo{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      ownerInfoResourceName,
					},
					Spec: smoothoperatorv1.OwnerInfoSpec{
						MetadataUrls: smoothoperatorv1.MetadataUrls{
							CSW: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
							},
							OpenSearch: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/opensearch/dut/{{identifier}}/OpenSearchDescription.xml",
							},
							HTML: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/catalog.search#/metadata/{{identifier}}",
							},
						},
						Atom: smoothoperatorv1.Atom{
							Author: smoothoperatormodel.Author{
								Name:  "pdok",
								Email: "pdokbeheer@kadaster.nl",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfo)).To(Succeed())
			}
		})

		AfterEach(func() {
			atomResource := &pdoknlv3.Atom{}
			atomResource.Name = typeNamespacedNameAtom.Name
			atomResource.Namespace = typeNamespacedNameAtom.Namespace
			err := k8sClient.Get(ctx, typeNamespacedNameAtom, atomResource)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Atom")
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, atomResource))).To(Succeed())

			ownerInfoResource := &smoothoperatorv1.OwnerInfo{}
			err = k8sClient.Get(ctx, typeNamespacedNameOwnerInfo, ownerInfoResource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance OwnerInfo")
			Expect(k8sClient.Delete(ctx, ownerInfoResource)).To(Succeed())
		})

		It("Should successfully create and delete its owned resources", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}
			By("Reconciling the Atom")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedNameAtom,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the finalizer")
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Finalizers).To(ContainElement(finalizerName))

			By("Reconciling the Atom again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedNameAtom,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for the owned resources to be created")
			Eventually(func() error {
				configMapName, err := getAtomGeneratorConfigMapName(ctx, atom)
				if err != nil {
					return err
				}
				expectedBareObjects := getExpectedBareObjectsForAtom(atom, configMapName)
				for _, d := range expectedBareObjects {
					err := k8sClient.Get(ctx, d.key, d.obj)
					if err != nil {
						return err
					}
				}
				return nil
			}, "10s", "1s").Should(Not(HaveOccurred()))

			By("Finding the ConfigMap name (with hash)")
			configMapName, err := getAtomGeneratorConfigMapName(ctx, atom)
			Expect(err).NotTo(HaveOccurred())

			By("Checking the status of the Atom")
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(atom.Status.Conditions)).To(BeEquivalentTo(1))
			Expect(atom.Status.Conditions[0].Status).To(BeEquivalentTo(metav1.ConditionTrue))

			By("Deleting the Atom")
			Expect(k8sClient.Delete(ctx, atom)).To(Succeed())

			By("Reconciling the Atom again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for the owned resources to be deleted")
			Eventually(func() error {
				expectedBareObjects := getExpectedBareObjectsForAtom(atom, configMapName)
				for _, d := range expectedBareObjects {
					err := k8sClient.Get(ctx, d.key, d.obj)
					if err == nil {
						return errors.New("expected " + getObjectFullName(k8sClient, d.obj) + " to not be found")
					}
					if !k8serrors.IsNotFound(err) {
						return err
					}
				}
				return nil
			}, "10s", "1s").Should(Not(HaveOccurred()))
		})

		It("Should successfully reconcile after a change in an owned resource", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom, checking the finalizer, and reconciling again")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			By("Getting the original Deployment")
			deployment := getBareDeployment(atom)
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())
			originalMinReadySeconds := deployment.Spec.MinReadySeconds

			// TODO: UNDERDEVELOPMENT
			log.Printf(" deployment.ObjectMeta.Labels[\"app\"]: %v", deployment.ObjectMeta.Labels["app"])
			log.Printf(" deployment.ObjectMeta.Labels[\"dataset\"]: %v", deployment.ObjectMeta.Labels["dataset"])
			log.Printf(" deployment.ObjectMeta.Labels[\"dataset-owner\"]: %v", deployment.ObjectMeta.Labels["dataset-owner"])
			log.Printf(" deployment.ObjectMeta.Labels[\"service-type\"]: %v", deployment.ObjectMeta.Labels["service-type"])
			Expect(int32(2)).Should(Equal(atomic.LoadInt32(deployment.Spec.Replicas)))

			log.Printf(" deployment.Spec.Replicas: %d", atomic.LoadInt32(deployment.Spec.Replicas))

			// TODO: END DEVELOPMENT

			By("Altering the Deployment")
			err = k8sClient.Patch(ctx, deployment, client.RawPatch(types.MergePatchType, []byte(
				`{"spec": {"minReadySeconds": 99}}`)))
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was altered")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(deployment.Spec.MinReadySeconds).To(BeEquivalentTo(99))
			}, "10s", "1s").Should(BeTrue())

			By("Reconciling the Atom again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the Deployment was restored")
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
				return Expect(err).NotTo(HaveOccurred()) &&
					Expect(deployment.Spec.MinReadySeconds).To(BeEquivalentTo(originalMinReadySeconds))
			}, "10s", "1s").Should(BeTrue())
		})
	})
})

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func getAtomGeneratorConfigMapName(ctx context.Context, atom *pdoknlv3.Atom) (string, error) {
	deployment := &appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: getBareDeployment(atom).GetName()}, deployment)
	if err != nil {
		return "", err
	}

	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Name == "config" && volume.ConfigMap != nil {
			return volume.ConfigMap.Name, nil
		}
	}
	return "", errors.New("atom generator configmap not found")
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
		{obj: &traefikiov1alpha1.Middleware{}, key: types.NamespacedName{Namespace: namespace, Name: getBareCorsHeadersMiddleware(atom).GetName()}},
		{obj: &corev1.Service{}, key: types.NamespacedName{Namespace: namespace, Name: getBareService(atom).GetName()}},
		{obj: &traefikiov1alpha1.IngressRoute{}, key: types.NamespacedName{Namespace: namespace, Name: getBareIngressRoute(atom).GetName()}},
		{obj: &policyv1.PodDisruptionBudget{}, key: types.NamespacedName{Namespace: namespace, Name: getBarePodDisruptionBudget(atom).GetName()}},
	}
	for index := range atom.GetIndexedDownloadLinks() {
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
						Lifecycle: smoothoperatormodel.Lifecycle{},
						Service: pdoknlv3.Service{
							ServiceMetadataLinks: pdoknlv3.MetadataLink{
								MetadataIdentifier: "7c5bbc80-d6f1-48d7-ba75-xxxxxxxxxxxx",
								Templates:          []string{"csw", "opensearch", "html"},
							},
						},
						DatasetFeeds: []pdoknlv3.DatasetFeed{
							{
								TechnicalName: "https://service.pdok.nl/test/atom/index.xml",
								Title:         "BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM",
								Subtitle:      "BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM",
								//Links:         []pdoknlv3.Link{},
								DatasetMetadataLinks: pdoknlv3.MetadataLink{
									MetadataIdentifier: "d893c05b-907e-47f2-9cbd-ceb08e68732c",
									Templates:          []string{"csw", "html"},
								},
								Author: smoothoperatormodel.Author{
									Name:  "owner",
									Email: "info@test.com",
								},
								SpatialDatasetIdentifierCode:      "d893c05b-907e-47f2-9cbd-ceb08e68732c",
								SpatialDatasetIdentifierNamespace: "http://www.pdok.nl",
								Entries: []pdoknlv3.Entry{
									{
										TechnicalName: "https://service.pdok.nl/test/atom/bro_geotechnisch_sondeeronderzoek_cpt_inspire_geharmoniseerd_geologie.xml",
										Title:         "BRO - Geotechnisch sondeeronderzoek (CPT) INSPIRE geharmoniseerd - Geologie",
										Content:       "Gegevens van geotechnisch sondeeronderzoek (kenset) zoals opgeslagen in de Basis Registratie Ondergrond (BRO).",
										DownloadLinks: []pdoknlv3.DownloadLink{
											{
												Data: "http://localazurite.blob.azurite/bucket/key1/dataset-1-file",
											},
										},
										Polygon: getTestPolygon(),
										Updated: &metav1.Time{Time: getUpdatedDate()},
										SRS: &pdoknlv3.SRS{
											Name: "Amersfoort / RD New",
											URI:  "https://www.opengis.net/def/crs/EPSG/0/28992",
										},
									},
								},
							},
						},
					},
				},
				ownerInfo: &smoothoperatorv1.OwnerInfo{
					Spec: smoothoperatorv1.OwnerInfoSpec{
						MetadataUrls: smoothoperatorv1.MetadataUrls{
							CSW: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
							},
							OpenSearch: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/opensearch/dut/{{identifier}}/OpenSearchDescription.xml",
							},
							HTML: smoothoperatorv1.MetadataURL{
								HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/catalog.search#/metadata/{{identifier}}",
							},
						},
					},
				},
			},
			wantConfig: "feeds:\n    - xmlname:\n        space: \"\"\n        local: \"\"\n      stylesheet: /atom/style/style.xsl\n      xmlns: http://www.w3.org/2005/Atom\n      georss: http://www.georss.org/georss\n      inspire_dls: http://inspire.ec.europa.eu/schemas/inspire_dls/1.0\n      lang: nl\n      id: /index.xml\n      title: \"\"\n      subtitle: \"\"\n      self: null\n      describedby: null\n      search: null\n      up: null\n      link:\n        - href: /index.xml\n          data: null\n          rel: self\n          type: application/atom+xml\n          hreflang: nl\n          length: \"\"\n          title: \"\"\n          version: null\n          time: null\n          bbox: null\n        - href: https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=7c5bbc80-d6f1-48d7-ba75-xxxxxxxxxxxx\n          data: null\n          rel: describedby\n          type: application/xml\n          hreflang: nl\n          length: \"\"\n          title: \"\"\n          version: null\n          time: null\n          bbox: null\n        - href: https://www.ngr.nl/geonetwork/opensearch/dut/7c5bbc80-d6f1-48d7-ba75-xxxxxxxxxxxx/OpenSearchDescription.xml\n          data: null\n          rel: search\n          type: application/xml\n          hreflang: nl\n          length: \"\"\n          title: \"\"\n          version: null\n          time: null\n          bbox: null\n        - href: https://www.ngr.nl/geonetwork/srv/dut/catalog.search#/metadata/7c5bbc80-d6f1-48d7-ba75-xxxxxxxxxxxx\n          data: null\n          rel: related\n          type: text/html\n          hreflang: nl\n          length: \"\"\n          title: \"\"\n          version: null\n          time: null\n          bbox: null\n      rights: \"\"\n      updated: \"2025-03-05T06:05:05+01:00\"\n      author:\n        name: \"\"\n        email: \"\"\n      entry:\n        - id: /https://service.pdok.nl/test/atom/index.xml.xml\n          title: BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM\n          content: \"\"\n          summary: BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM\n          link:\n            - href: https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=7c5bbc80-d6f1-48d7-ba75-xxxxxxxxxxxx\n              data: null\n              rel: describedby\n              type: application/xml\n              hreflang: nl\n              length: \"\"\n              title: \"\"\n              version: null\n              time: null\n              bbox: null\n            - href: /https://service.pdok.nl/test/atom/index.xml.xml\n              data: null\n              rel: alternate\n              type: application/atom+xml\n              hreflang: null\n              length: \"\"\n              title: BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM\n              version: null\n              time: null\n              bbox: null\n          rights: \"\"\n          updated: \"2025-03-05T06:05:05+01:00\"\n          polygon: 1 1 1 2 2 2 2 1 1 1\n          category:\n            - term: https://www.opengis.net/def/crs/EPSG/0/28992\n              label: Amersfoort / RD New\n          spatial_dataset_identifier_code: d893c05b-907e-47f2-9cbd-ceb08e68732c\n          spatial_dataset_identifier_namespace: http://www.pdok.nl\n    - xmlname:\n        space: \"\"\n        local: \"\"\n      stylesheet: /atom/style/style.xsl\n      xmlns: \"\"\n      georss: \"\"\n      inspire_dls: \"\"\n      lang: nl\n      id: /https://service.pdok.nl/test/atom/index.xml.xml\n      title: BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM\n      subtitle: BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM\n      self: null\n      describedby: null\n      search: null\n      up: null\n      link:\n        - href: /https://service.pdok.nl/test/atom/index.xml.xml\n          data: null\n          rel: self\n          type: \"\"\n          hreflang: null\n          length: \"\"\n          title: \"\"\n          version: null\n          time: null\n          bbox: null\n        - href: /index.xml\n          data: null\n          rel: up\n          type: application/atom+xml\n          hreflang: null\n          length: \"\"\n          title: Top Atom Download Service Feed\n          version: null\n          time: null\n          bbox: null\n        - href: https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=7c5bbc80-d6f1-48d7-ba75-xxxxxxxxxxxx\n          data: null\n          rel: describedby\n          type: text.html\n          hreflang: null\n          length: \"\"\n          title: \"\"\n          version: null\n          time: null\n          bbox: null\n        - href: https://www.ngr.nl/geonetwork/srv/dut/catalog.search#/metadata/7c5bbc80-d6f1-48d7-ba75-xxxxxxxxxxxx\n          data: null\n          rel: \"\"\n          type: text.html\n          hreflang: null\n          length: \"\"\n          title: NGR pagina voor deze dataset\n          version: null\n          time: null\n          bbox: null\n      rights: \"\"\n      author:\n        name: owner\n        email: info@test.com\n      entry:\n        - id: /https://service.pdok.nl/test/atom/bro_geotechnisch_sondeeronderzoek_cpt_inspire_geharmoniseerd_geologie.xml.xml\n          title: BRO - Geotechnisch sondeeronderzoek (CPT) INSPIRE geharmoniseerd - Geologie\n          content: Gegevens van geotechnisch sondeeronderzoek (kenset) zoals opgeslagen in de Basis Registratie Ondergrond (BRO).\n          summary: BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM\n          link:\n            - href: /downloads/dataset-1-file\n              data: /http://localazurite.blob.azurite/bucket/key1/dataset-1-file\n              rel: alternate\n              type: \"\"\n              hreflang: null\n              length: \"\"\n              title: BRO - Geotechnisch sondeeronderzoek (CPT) INSPIRE geharmoniseerd - Geologie-dataset-1-file\n              version: null\n              time: null\n              bbox: null\n          rights: \"\"\n          updated: \"2025-03-05T06:05:05+01:00\"\n          polygon: 1 1 1 2 2 2 2 1 1 1\n          category:\n            - term: https://www.opengis.net/def/crs/EPSG/0/28992\n              label: Amersfoort / RD New\n          spatial_dataset_identifier_code: \"\"\n          spatial_dataset_identifier_namespace: \"\"\n",
			wantErr:    false,
		},
		//{
		//	name: "succesfull_scenario_03",
		//	args: args{
		//		atom: &pdoknlv3.Atom{
		//			Spec: pdoknlv3.AtomSpec{
		//				Lifecycle: smoothoperatormodel.Lifecycle{},
		//				Service: pdoknlv3.Service{
		//					ServiceMetadataLinks: pdoknlv3.MetadataLink{
		//						MetadataIdentifier: "7c5bbc80-d6f1-48d7-ba75-xxxxxxxxxxxx",
		//						Templates:          []string{"csw", "opensearch", "html"},
		//					},
		//				},
		//				DatasetFeeds: []pdoknlv3.DatasetFeed{
		//					{
		//						TechnicalName: "https://service.pdok.nl/test/atom/index.xml",
		//						Title:         "BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM",
		//						Subtitle:      "BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM",
		//						//Links:         []pdoknlv3.Link{},
		//						DatasetMetadataLinks: pdoknlv3.MetadataLink{
		//							MetadataIdentifier: "d893c05b-907e-47f2-9cbd-ceb08e68732c",
		//							Templates:          []string{"csw", "html"},
		//						},
		//						Author: smoothoperatormodel.Author{
		//							Name:  "owner",
		//							Email: "info@test.com",
		//						},
		//						SpatialDatasetIdentifierCode:      "d893c05b-907e-47f2-9cbd-ceb08e68732c",
		//						SpatialDatasetIdentifierNamespace: "http://www.pdok.nl",
		//						Entries: []pdoknlv3.Entry{
		//							{
		//								TechnicalName: "https://service.pdok.nl/test/atom/bro_geotechnisch_sondeeronderzoek_cpt_inspire_geharmoniseerd_geologie.xml",
		//								Title:         "BRO - Geotechnisch sondeeronderzoek (CPT) INSPIRE geharmoniseerd - Geologie",
		//								Content:       "Gegevens van geotechnisch sondeeronderzoek (kenset) zoals opgeslagen in de Basis Registratie Ondergrond (BRO).",
		//								DownloadLinks: []pdoknlv3.DownloadLink{
		//									{
		//										Data: "http://localazurite.blob.azurite/bucket/key1/dataset-1-file",
		//									},
		//								},
		//								Polygon: getTestPolygon(),
		//								Updated: &metav1.Time{Time: getUpdatedDate()},
		//								SRS: &pdoknlv3.SRS{
		//									Name: "Amersfoort / RD New",
		//									URI:  "https://www.opengis.net/def/crs/EPSG/0/28992",
		//								},
		//							},
		//						},
		//					},
		//				},
		//			},
		//		},
		//		ownerInfo: &smoothoperatorv1.OwnerInfo{
		//			Spec: smoothoperatorv1.OwnerInfoSpec{
		//				MetadataUrls: smoothoperatorv1.MetadataUrls{
		//					CSW: smoothoperatorv1.MetadataURL{
		//						HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
		//					},
		//					OpenSearch: smoothoperatorv1.MetadataURL{
		//						HrefTemplate: "https://www.ngr.nl/geonetwork/opensearch/dut/{{identifier}}/OpenSearchDescription.xml",
		//					},
		//					HTML: smoothoperatorv1.MetadataURL{
		//						HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/catalog.search#/metadata/{{identifier}}",
		//					},
		//				},
		//			},
		//		},
		//	},
		//	wantConfig: readTestFile("generatorConfigData_testdata.yaml"),
		//	wantErr:    false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConfig, err := getGeneratorConfig(tt.args.atom, tt.args.ownerInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("getGeneratorConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if removeSpace(gotConfig) != removeSpace(tt.wantConfig) {
				t.Errorf("getGeneratorConfig() gotConfig = %v, want %v", gotConfig, tt.wantConfig)
			}
		})
	}
}

/*func readTestFile(fileName string) string {
	dat, _ := os.ReadFile(fileName)

	return string(dat)
}*/

func getTestPolygon() *pdoknlv3.Polygon {
	return &pdoknlv3.Polygon{
		BBox: smoothoperatormodel.BBox{
			MinX: "1",
			MinY: "1",
			MaxX: "2",
			MaxY: "2",
		},
	}
}

func getUpdatedDate() time.Time {
	return metav1.Date(2025, time.March, 5, 5, 5, 5, 0, time.UTC).Local()
}

func removeSpace(s string) string {
	rr := make([]rune, 0, len(s))
	for _, r := range s {
		if !unicode.IsSpace(r) {
			rr = append(rr, r)
		}
	}
	return string(rr)
}
