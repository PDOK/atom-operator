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
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
	"unicode"

	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	policyv1 "k8s.io/api/policy/v1"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

var _ = Describe("Atom Controller", func() {
	Context("When reconciling a resource", func() {

		ctx := context.Background()

		pdoknlv3.SetHost("localhost")
		// Setup variables for unique Atom resource per It node
		counter := 1
		var fullAtom pdoknlv3.Atom
		var typeNamespacedNameAtom types.NamespacedName

		atom := &pdoknlv3.Atom{}

		typeNamespacedNameOwnerInfo := types.NamespacedName{
			Namespace: namespace,
			Name:      ownerInfoResourceName,
		}
		ownerInfo := &smoothoperatorv1.OwnerInfo{}

		BeforeEach(func() {
			// Create a unique Atom resource for every It node to prevent unexpected resource state caused by finalizers
			fullAtom = getUniqueFullAtom(counter)
			typeNamespacedNameAtom = getUniqueAtomTypeNamespacedName(counter)
			counter++

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
						return errors.New("expected " + smoothoperatorutils.GetObjectFullName(k8sClient, d.obj) + " to not be found")
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

		It("Should create correct deployment manifest.", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom and checking the deployment manifest")
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

			Expect(deployment.ObjectMeta.Labels["app"]).Should(Equal("atom-service"))
			Expect(deployment.ObjectMeta.Labels["dataset"]).Should(Equal("test-dataset"))
			Expect(deployment.ObjectMeta.Labels["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(deployment.ObjectMeta.Labels["service-type"]).Should(Equal("atom"))
			Expect(deployment.ObjectMeta.Namespace).Should(Equal("default"))
			Expect(atomic.LoadInt32(deployment.Spec.Replicas)).Should(Equal(int32(2)))
			Expect(atomic.LoadInt32(deployment.Spec.RevisionHistoryLimit)).Should(Equal(int32(1)))

			TestStrategy := appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 4},
				},
			}
			Expect(TestStrategy.Type).Should(Equal(deployment.Spec.Strategy.Type))
			Expect(TestStrategy.RollingUpdate.MaxUnavailable).Should(Equal(deployment.Spec.Strategy.RollingUpdate.MaxUnavailable))
			Expect(TestStrategy.RollingUpdate.MaxSurge).Should(Equal(deployment.Spec.Strategy.RollingUpdate.MaxSurge))
			Expect(deployment.Spec.Selector.MatchLabels["app"]).Should(Equal("atom-service"))
			Expect(deployment.Spec.Selector.MatchLabels["dataset"]).Should(Equal("test-dataset"))
			Expect(deployment.Spec.Selector.MatchLabels["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(deployment.Spec.Selector.MatchLabels["service-type"]).Should(Equal("atom"))

			log.Printf("deployment.Spec.Template.ObjectMeta.Annotations[\"cluster-autoscaler.kubernetes.io/safe-to-evict\"]: %v", deployment.Spec.Template.ObjectMeta.Annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"])

			/* TODO: de controller vult de volgende niet. Is dat ok?
			Expect(nil).Should(Equal(deployment.Spec.Template.ObjectMeta.Annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"]))

						cluster-autoscaler.kubernetes.io/safe-to-evict: 'true'
						kubectl.kubernetes.io/default-container: atom-service
						priority.version-checker.io/atom-service: "8"
			*/

			Expect(deployment.Spec.Template.ObjectMeta.Labels["app"]).Should(Equal("atom-service"))
			Expect(deployment.Spec.Template.ObjectMeta.Labels["dataset"]).Should(Equal("test-dataset"))
			Expect(deployment.Spec.Template.ObjectMeta.Labels["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(deployment.Spec.Template.ObjectMeta.Labels["service-type"]).Should(Equal("atom"))

			Expect(deployment.Spec.Template.Spec.Containers[0].Name).Should(Equal("atom-service"))

			Expect(deployment.Spec.Template.Spec.Containers[0].Ports[0].Name).Should(Equal("atom-service"))
			Expect(deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort).Should(Equal(int32(80)))
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).Should(Equal(testImageName2))
			Expect(deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))

			httpGet := &corev1.HTTPGetAction{
				Path:   "/index.xml",
				Port:   intstr.FromInt32(atomPortNr),
				Scheme: corev1.URISchemeHTTP,
			}
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet).Should(Equal(httpGet))
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path).Should(Equal(httpGet.Path))
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port).Should(Equal(httpGet.Port))
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Scheme).Should(Equal(httpGet.Scheme))
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.InitialDelaySeconds).Should(Equal(int32(5)))
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.PeriodSeconds).Should(Equal(int32(10)))
			Expect(deployment.Spec.Template.Spec.Containers[0].LivenessProbe.TimeoutSeconds).Should(Equal(int32(5)))

			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet).Should(Equal(httpGet))
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.InitialDelaySeconds).Should(Equal(int32(5)))
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.PeriodSeconds).Should(Equal(int32(10)))
			Expect(deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.TimeoutSeconds).Should(Equal(int32(5)))

			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String()).Should(Equal("64M"))
			Expect(deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String()).Should(Equal("10m"))

			expectedVolumeMounts := []corev1.VolumeMount{
				{Name: "socket", MountPath: "/tmp", ReadOnly: false},
				{Name: "data", MountPath: "var/www"},
			}
			Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts).Should(Equal(expectedVolumeMounts))

			Expect(deployment.Spec.Template.Spec.InitContainers[0].Name).Should(Equal("atom-generator"))
			Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).Should(Equal(testImageName1))
			Expect(deployment.Spec.Template.Spec.InitContainers[0].ImagePullPolicy).Should(Equal(corev1.PullIfNotPresent))
			Expect(deployment.Spec.Template.Spec.InitContainers[0].Command).Should(Equal([]string{"./atom"}))
			Expect(deployment.Spec.Template.Spec.InitContainers[0].Args[0]).Should(Equal("-f=/srv/config/values.yaml"))
			Expect(deployment.Spec.Template.Spec.InitContainers[0].Args[1]).Should(Equal("-o=/srv/data"))

			VolumeMounts := []corev1.VolumeMount{
				{Name: "data", MountPath: srvDir + "/data"},
				{Name: "config", MountPath: srvDir + "/config"},
			}
			Expect(deployment.Spec.Template.Spec.InitContainers[0].VolumeMounts).Should(Equal(VolumeMounts))

			testEmptyDir := &corev1.EmptyDirVolumeSource{}
			Expect(deployment.Spec.Template.Spec.Volumes[0].Name).Should(Equal("data"))
			Expect(deployment.Spec.Template.Spec.Volumes[0].EmptyDir).Should(Equal(testEmptyDir))

			Expect(deployment.Spec.Template.Spec.Volumes[1].Name).Should(Equal("socket"))
			Expect(deployment.Spec.Template.Spec.Volumes[1].EmptyDir).Should(Equal(testEmptyDir))
			Expect(deployment.Spec.Template.Spec.Volumes[2].Name).Should(Equal("config"))
			Expect(deployment.Spec.Template.Spec.Volumes[2].ConfigMap.Name).Should(ContainSubstring("test-atom-3-atom-service-"))
		})

		It("Should create correct configmap-atom-generator manifest.", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom and checking the configmap")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			configMap := getBareConfigMap(atom)
			configMapName, err := getAtomConfigMapNameFromClient(ctx, atom)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: atom.GetNamespace(), Name: configMapName}, configMap)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			testTure := true
			Expect(configMap.Name).Should(Equal(configMapName))
			Expect(configMap.ObjectMeta.Name).Should(Equal(configMapName))
			Expect(configMap.ObjectMeta.Namespace).Should(Equal(atom.Namespace))
			Expect(configMap.Immutable).Should(Equal(&testTure))
			Expect(len(configMap.Labels)).Should(Equal(4))
			Expect(configMap.Labels["app"]).Should(Equal("atom-service"))
			Expect(configMap.Labels["dataset"]).Should(Equal("test-dataset"))
			Expect(configMap.Labels["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(configMap.Labels["service-type"]).Should(Equal("atom"))

			Expect(configMap.Data["values.yaml"]).Should(ContainSubstring("feeds:"))
			Expect(configMap.Data["values.yaml"]).Should(ContainSubstring("rel: self"))
			Expect(configMap.Data["values.yaml"]).Should(ContainSubstring("href: https://my.test-resource.test/test-datasetowner/test-dataset/atom/index.xml"))
			Expect(configMap.Data["values.yaml"]).Should(ContainSubstring("type: application/atom+xml"))
			Expect(configMap.Data["values.yaml"]).Should(ContainSubstring("title: test title"))
		})

		It("Should create correct IngressRoute manifest.", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom and checking the IngressRoute")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			ingressRoute := getBareIngressRoute(atom)
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(ingressRoute), ingressRoute)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(ingressRoute.Name).Should(Equal(atom.Name))
			Expect(ingressRoute.Namespace).Should(Equal(atom.Namespace))
			Expect(ingressRoute.Labels["app"]).Should(Equal(atom.Labels["app"]))
			Expect(ingressRoute.Labels["dataset"]).Should(Equal(atom.Labels["dataset"]))
			Expect(ingressRoute.Labels["dataset-owner"]).Should(Equal(atom.Labels["dataset-owner"]))
			Expect(ingressRoute.Labels["service-type"]).Should(Equal(atom.Labels["service-type"]))

			Expect(ingressRoute.Spec.Routes[0].Kind).Should(Equal("Rule"))
			Expect(ingressRoute.Spec.Routes[0].Match).Should(Equal("Host(`localhost`) && Path(`/test-datasetowner/test-dataset/atom/index.xml`)"))
			Expect(ingressRoute.Spec.Routes[0].Priority).Should(Equal(0))
			Expect(len(ingressRoute.Spec.Routes[0].Services)).Should(Equal(1))
			Expect(ingressRoute.Spec.Routes[0].Services[0].Name).Should(Equal("test-atom-5-atom"))
			Expect(ingressRoute.Spec.Routes[0].Services[0].Kind).Should(Equal("Service"))
			Expect(ingressRoute.Spec.Routes[0].Services[0].Namespace).Should(Equal(""))
			Expect(ingressRoute.Spec.Routes[0].Services[0].Port.IntVal).Should(Equal(int32(80)))
			Expect(ingressRoute.Spec.Routes[0].Middlewares[0].Name).Should(Equal("test-atom-5-atom-strip-prefix"))
			Expect(ingressRoute.Spec.Routes[0].Middlewares[0].Namespace).Should(Equal("default"))
			Expect(ingressRoute.Spec.Routes[0].Middlewares[1].Name).Should(Equal("test-atom-5-atom-cors-headers"))
			Expect(ingressRoute.Spec.Routes[0].Middlewares[1].Namespace).Should(Equal("default"))

			Expect(ingressRoute.Spec.Routes[1].Kind).Should(Equal("Rule"))
			Expect(ingressRoute.Spec.Routes[1].Match).Should(Equal("Host(`localhost`) && PathPrefix(`/test-datasetowner/test-dataset/atom/downloads/`)"))
			Expect(ingressRoute.Spec.Routes[1].Priority).Should(Equal(0))
			Expect(len(ingressRoute.Spec.Routes[1].Services)).Should(Equal(1))
			Expect(ingressRoute.Spec.Routes[1].Services[0].Name).Should(Equal("azure-storage"))
			Expect(ingressRoute.Spec.Routes[1].Services[0].Kind).Should(Equal("Service"))
			Expect(ingressRoute.Spec.Routes[1].Services[0].Namespace).Should(Equal(""))
			Expect(ingressRoute.Spec.Routes[1].Services[0].Port.StrVal).Should(Equal(intstr.FromString("azure-storage").StrVal))
			Expect(ingressRoute.Spec.Routes[1].Middlewares[0].Name).Should(Equal("test-atom-5-atom-cors-headers"))
			Expect(ingressRoute.Spec.Routes[1].Middlewares[0].Namespace).Should(Equal("default"))
			Expect(ingressRoute.Spec.Routes[1].Middlewares[1].Name).Should(Equal("test-atom-5-atom-downloads-0"))
			Expect(ingressRoute.Spec.Routes[1].Middlewares[1].Namespace).Should(Equal("default"))

			Expect(ingressRoute.Spec.Routes[2].Kind).Should(Equal("Rule"))
			Expect(ingressRoute.Spec.Routes[2].Match).Should(Equal("Host(`localhost`) && Path(`/test-datasetowner/test-dataset/atom/test-technical-name.xml`)"))
			Expect(ingressRoute.Spec.Routes[2].Priority).Should(Equal(0))
			Expect(len(ingressRoute.Spec.Routes[2].Services)).Should(Equal(1))
			Expect(ingressRoute.Spec.Routes[2].Services[0].Name).Should(Equal("test-atom-5-atom"))
			Expect(ingressRoute.Spec.Routes[2].Services[0].Kind).Should(Equal("Service"))
			Expect(ingressRoute.Spec.Routes[2].Services[0].Namespace).Should(Equal(""))
			Expect(ingressRoute.Spec.Routes[2].Services[0].Port.IntVal).Should(Equal(int32(80)))
			Expect(ingressRoute.Spec.Routes[2].Middlewares[0].Name).Should(Equal("test-atom-5-atom-strip-prefix"))
			Expect(ingressRoute.Spec.Routes[2].Middlewares[0].Namespace).Should(Equal("default"))
			Expect(ingressRoute.Spec.Routes[2].Middlewares[1].Name).Should(Equal("test-atom-5-atom-cors-headers"))
			Expect(ingressRoute.Spec.Routes[2].Middlewares[1].Namespace).Should(Equal("default"))

		})

		It("Should create correct middlewareCorsHeaders manifest.", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom and checking the middlewareCorsHeaders")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			middlewareCorsHeaders := getBareCorsHeadersMiddleware(atom)
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(middlewareCorsHeaders), middlewareCorsHeaders)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(middlewareCorsHeaders.Name).Should(Equal("test-atom-6-atom-cors-headers"))
			Expect(middlewareCorsHeaders.Namespace).Should(Equal("default"))
			Expect(middlewareCorsHeaders.Labels["app"]).Should(Equal("atom-service"))
			Expect(middlewareCorsHeaders.Labels["dataset"]).Should(Equal("test-dataset"))
			Expect(middlewareCorsHeaders.Labels["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(middlewareCorsHeaders.Labels["service-type"]).Should(Equal("atom"))
			Expect(middlewareCorsHeaders.Spec.Headers.FrameDeny).Should(Equal(true))
			Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Headers"]).Should(Equal("Content-Type"))
			Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Method"]).Should(Equal("GET, HEAD, OPTIONS"))
			Expect(middlewareCorsHeaders.Spec.Headers.CustomResponseHeaders["Access-Control-Allow-Origin"]).Should(Equal("*"))
		})

		It("Should create correct middlewareStripPrefix.", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom and checking the middlewareStripPrefix")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			middlewareStripPrefix := getBareStripPrefixMiddleware(atom)
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(middlewareStripPrefix), middlewareStripPrefix)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(middlewareStripPrefix.Name).Should(Equal("test-atom-7-atom-strip-prefix"))
			Expect(middlewareStripPrefix.Namespace).Should(Equal("default"))
			Expect(middlewareStripPrefix.Labels["app"]).Should(Equal("atom-service"))
			Expect(middlewareStripPrefix.Labels["dataset"]).Should(Equal("test-dataset"))
			Expect(middlewareStripPrefix.Labels["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(middlewareStripPrefix.Labels["service-type"]).Should(Equal("atom"))
			Expect(middlewareStripPrefix.Spec.StripPrefix.Prefixes[0]).Should(Equal("/test-datasetowner/test-dataset/atom/"))
		})

		It("Should create correct middleware atom download.", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom and checking the middlewareStripPrefix")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			downloadMiddlewareArray := getDownloadMiddlewareArray(ctx, atom)
			for i := 0; i < len(downloadMiddlewareArray); i++ {
				Expect(downloadMiddlewareArray[i].Name).Should(Equal("test-atom-8-atom-downloads-" + strconv.Itoa(i)))
				Expect(downloadMiddlewareArray[i].Namespace).Should(Equal("default"))
				Expect(downloadMiddlewareArray[i].Labels["app"]).Should(Equal("atom-service"))
				Expect(downloadMiddlewareArray[i].Labels["dataset"]).Should(Equal("test-dataset"))
				Expect(downloadMiddlewareArray[i].Labels["dataset-owner"]).Should(Equal("test-datasetowner"))
				Expect(downloadMiddlewareArray[i].Labels["service-type"]).Should(Equal("atom"))
				Expect(downloadMiddlewareArray[i].Spec.ReplacePathRegex.Regex).Should(Equal("^/test-datasetowner/test-dataset/atom/downloads/(dataset-1-file)"))
				Expect(downloadMiddlewareArray[i].Spec.ReplacePathRegex.Replacement).Should(Equal("/http://localazurite.blob.azurite/bucket/key1/$1"))

			}
		})

		It("Should create correct podDisruptionBudget.", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom and checking the podDisruptionBudget")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			podDisruptionBudget := getBarePodDisruptionBudget(atom)
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(podDisruptionBudget), podDisruptionBudget)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(podDisruptionBudget.Name).Should(Equal("test-atom-9-atom-pdb"))
			Expect(podDisruptionBudget.Namespace).Should(Equal("default"))
			Expect(podDisruptionBudget.Labels["app"]).Should(Equal("atom-service"))
			Expect(podDisruptionBudget.Labels["dataset"]).Should(Equal("test-dataset"))
			Expect(podDisruptionBudget.Labels["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(podDisruptionBudget.Labels["service-type"]).Should(Equal("atom"))
			Expect(podDisruptionBudget.Spec.MaxUnavailable).Should(Equal(&intstr.IntOrString{IntVal: 1}))
			Expect(podDisruptionBudget.Spec.Selector.MatchLabels["app"]).Should(Equal("atom-service"))
			Expect(podDisruptionBudget.Spec.Selector.MatchLabels["dataset"]).Should(Equal("test-dataset"))
			Expect(podDisruptionBudget.Spec.Selector.MatchLabels["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(podDisruptionBudget.Spec.Selector.MatchLabels["service-type"]).Should(Equal("atom"))
		})

		It("Should create correct Service atom.", func() {
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			By("Reconciling the Atom and checking the Service atom")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, typeNamespacedNameAtom, atom)
			Expect(err).NotTo(HaveOccurred())
			Expect(atom.Finalizers).To(ContainElement(finalizerName))
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedNameAtom})
			Expect(err).NotTo(HaveOccurred())

			service := getBareService(atom)
			Eventually(func() bool {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(service), service)
				return Expect(err).NotTo(HaveOccurred())
			}, "10s", "1s").Should(BeTrue())

			Expect(service.Name).Should(Equal("test-atom-10-atom"))
			Expect(service.Namespace).Should(Equal("default"))
			Expect(service.Labels["app"]).Should(Equal("atom-service"))
			Expect(service.Labels["dataset"]).Should(Equal("test-dataset"))
			Expect(service.Labels["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(service.Labels["service-type"]).Should(Equal("atom"))
			Expect(service.Spec.Ports[0].Name).Should(Equal("atom-service"))
			Expect(service.Spec.Ports[0].Port).Should(Equal(int32(80)))
			Expect(service.Spec.Ports[0].Protocol).Should(Equal(corev1.ProtocolTCP))
			Expect(service.Spec.Selector["app"]).Should(Equal("atom-service"))
			Expect(service.Spec.Selector["dataset"]).Should(Equal("test-dataset"))
			Expect(service.Spec.Selector["dataset-owner"]).Should(Equal("test-datasetowner"))
			Expect(service.Spec.Selector["service-type"]).Should(Equal("atom"))
		})
	})
})

func getDownloadMiddlewareArray(ctx context.Context, atom metav1.Object) []*traefikiov1alpha1.Middleware {
	var downloadMiddlewareArray []*traefikiov1alpha1.Middleware
	var err error
	index := int8(0)
	for err == nil {
		middlewareDownloadLink := getBareDownloadLinkMiddleware(atom, index)
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(middlewareDownloadLink), middlewareDownloadLink)
		if err != nil {
			break
		}
		downloadMiddlewareArray = append(downloadMiddlewareArray, middlewareDownloadLink)
		index++
	}
	log.Printf("getDownloadMiddlewareArray: %v", len(downloadMiddlewareArray))
	return downloadMiddlewareArray
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
											Content:       smoothoperatorutils.Pointer("Gegevens van geotechnisch sondeeronderzoek (kenset) zoals opgeslagen in de Basis Registratie Ondergrond (BRO)."),
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
			wantConfig: readTestFile("generator_config_test_data/scenario-2.yaml"),
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

func getUniqueFullAtom(counter int) pdoknlv3.Atom {
	return pdoknlv3.Atom{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      getUniqueAtomResourceName(counter),
			Labels: map[string]string{
				"app":           "atom-service",
				"dataset":       "test-dataset",
				"dataset-owner": "test-datasetowner",
				"service-type":  "atom",
			},
		},
		Spec: pdoknlv3.AtomSpec{
			Lifecycle: smoothoperatormodel.Lifecycle{
				TTLInDays: smoothoperatorutils.Pointer(int32(999)),
			},
			Service: pdoknlv3.Service{
				BaseURL:      "https://my.test-resource.test/test-datasetowner/test-dataset/atom",
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
			},
			PodSpecPatch: &corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name: "atom-generator",
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
}

func getUniqueAtomTypeNamespacedName(counter int) types.NamespacedName {
	return types.NamespacedName{
		Name:      getUniqueAtomResourceName(counter),
		Namespace: namespace,
	}
}

func getUniqueAtomResourceName(counter int) string {
	return fmt.Sprintf("%s-%v", atomResourceName, counter)
}

func readTestFile(fileName string) string {
	dat, _ := os.ReadFile(fileName)

	return string(dat)
}

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
	return metav1.Date(2025, time.March, 5, 5, 5, 5, 0, time.UTC).UTC()
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
