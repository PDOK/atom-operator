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
	"os"
	"testing"

	"github.com/pdok/smooth-operator/model"

	"github.com/google/go-cmp/cmp"
	"github.com/pdok/atom-generator/feeds"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
	atomyaml "sigs.k8s.io/yaml/goyaml.v3"

	policyv1 "k8s.io/api/policy/v1"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatorutils "github.com/pdok/smooth-operator/pkg/util"
	smoothoperatorvalidation "github.com/pdok/smooth-operator/pkg/validation"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

const (
	testImageName1 = "test.test/image:test1"
	testImageName2 = "test.test/image:test2"
)

var _ = Describe("Testing Atom Controller", func() {

	Context("fail", func() {
		It("fails", func() {
			Fail("failed")
		})
	})

	Context("Testing Mutate functions for Minimal Atom", func() {
		testAtomMutates("minimal")
	})

	Context("Testing Mutate functions for Maximal Atom", func() {

		testAtomMutates("maximum")

	})

	Context("When reconciling a resource", func() {

		ctx := context.Background()

		inputPath := testPath("maximum") + "input/"

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
			data, err := os.ReadFile(inputPath + "atom.yaml")
			Expect(err).NotTo(HaveOccurred())
			err = yaml.UnmarshalStrict(data, &testAtom)
			Expect(err).NotTo(HaveOccurred())
			Expect(testAtom.Name).Should(Equal("maximum"))

			objectKeyAtom = types.NamespacedName{
				Namespace: testAtom.GetNamespace(),
				Name:      testAtom.GetName(),
			}

			data, err = os.ReadFile(inputPath + "ownerinfo.yaml")
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

		It("Respects the TTL of the WMS", func() {
			By("Creating a new resource for the Kind WMS")
			controllerReconciler := &AtomReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				AtomGeneratorImage: testImageName1,
				LighttpdImage:      testImageName2,
			}

			ttlName := testAtom.GetName() + "-ttl"
			ttlAtom := testAtom.DeepCopy()
			ttlAtom.Name = ttlName
			ttlAtom.Spec.Lifecycle = &model.Lifecycle{TTLInDays: smoothoperatorutils.Pointer(int32(0))}
			objectKeyTTLAtom := client.ObjectKeyFromObject(ttlAtom)

			err := k8sClient.Get(ctx, objectKeyTTLAtom, ttlAtom)
			Expect(client.IgnoreNotFound(err)).To(Not(HaveOccurred()))
			if err != nil && apierrors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, ttlAtom)).To(Succeed())
			}

			// Reconcile
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKeyTTLAtom})
			Expect(err).To(Not(HaveOccurred()))

			// Check the WMS cannot be found anymore
			Eventually(func() bool {
				err = k8sClient.Get(ctx, objectKeyTTLAtom, ttlAtom)
				return apierrors.IsNotFound(err)
			}, "10s", "1s").Should(BeTrue())

			// Not checking owned resources because the test env does not do garbage collection
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

			// the testEnv does not do garbage collection (https://book.kubebuilder.io/reference/envtest#testing-considerations)
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

			filepath := "input/atom.yaml"
			testCases := []string{
				testPath("minimal") + filepath,
				testPath("maximum") + filepath,
			}

			for _, test := range testCases {
				yamlInput, err := readTestFile(test)
				Expect(err).NotTo(HaveOccurred())

				err = smoothoperatorvalidation.ValidateSchema(yamlInput)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})

func testAtomMutates(name string) {

	pdoknlv3.SetBlobEndpoint("http://localazurite.blob.azurite")

	var reconciler AtomReconciler

	inputPath := testPath(name) + "input/"
	outputPath := testPath(name) + "expected-output/"

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

		data, err := os.ReadFile(inputPath + "atom.yaml")
		Expect(err).NotTo(HaveOccurred())
		err = yaml.UnmarshalStrict(data, &atom)
		Expect(err).NotTo(HaveOccurred())
		Expect(atom.Name).Should(Equal(name))

		data, err = os.ReadFile(inputPath + "ownerinfo.yaml")
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

		var expectedValues, gottenValues feeds.Feeds
		err = atomyaml.Unmarshal([]byte(expected.Data["values.yaml"]), &expectedValues)
		Expect(err).NotTo(HaveOccurred())
		err = atomyaml.Unmarshal([]byte(result.Data["values.yaml"]), &gottenValues)
		Expect(err).NotTo(HaveOccurred())

		valuesDiff := cmp.Diff(expectedValues, gottenValues)
		if valuesDiff != "" {
			Fail(valuesDiff)
		}

		expected.Data["values.yaml"] = `"feed": []`
		result.Data["values.yaml"] = `"feed": []`
		diff := cmp.Diff(expected, *result)

		if diff != "" {
			Fail(diff)
		}

	})

	It("Should generate a Deployment correctly", func() {
		testMutate("Deployment", getBareDeployment(&atom), outputPath+"deployment.yaml", func(d *appsv1.Deployment) error {
			return reconciler.mutateDeployment(&atom, d, name+"-atom-generator")
		})
	})

	It("Should generate a correct Service", func() {
		testMutate("Service", getBareService(&atom), outputPath+"service.yaml", func(s *corev1.Service) error {
			return reconciler.mutateService(&atom, s)
		})
	})

	It("Should generate a correct Prefix Strip Middleware", func() {
		testMutate("Prefix Strip Middleware", getBareStripPrefixMiddleware(&atom), outputPath+"middleware-prefixstrip.yaml", func(m *traefikiov1alpha1.Middleware) error {
			return reconciler.mutateStripPrefixMiddleware(&atom, m)
		})
	})

	It("Should generate a correct Headers Middleware", func() {
		testMutate("Headers Middleware", getBareHeadersMiddleware(&atom), outputPath+"middleware-headers.yaml", func(m *traefikiov1alpha1.Middleware) error {
			return reconciler.mutateHeadersMiddleware(&atom, m, "default-src 'self';")
		})
	})

	It("Should generate a correct Download Middlewares", func() {
		for prefix, group := range getDownloadLinkGroups(atom.GetDownloadLinks()) {
			testMutate(fmt.Sprintf("Download Middleware %d", *group.index), getBareDownloadLinkMiddleware(&atom, *group.index), outputPath+fmt.Sprintf("middleware-downloads-%d.yaml", *group.index), func(m *traefikiov1alpha1.Middleware) error {
				return reconciler.mutateDownloadLinkMiddleware(&atom, prefix, group.files, m)
			})
		}

	})

	It("Should generate a correct IngressRoute", func() {
		testMutate("IngressRoute", getBareIngressRoute(&atom), outputPath+"ingressroute.yaml", func(i *traefikiov1alpha1.IngressRoute) error {
			return reconciler.mutateIngressRoute(&atom, i)
		})
	})

	It("Should generate a correct PodDisruptionBudget", func() {
		testMutate("PodDisruptionBudget", getBarePodDisruptionBudget(&atom), outputPath+"poddisruptionbudget.yaml", func(p *policyv1.PodDisruptionBudget) error {
			return reconciler.mutatePodDisruptionBudget(&atom, p)
		})
	})

}

func testPath(name string) string {
	return fmt.Sprintf("test_data/%s-atom/", name)
}

func testMutate[T any](kind string, result *T, expectedFile string, mutate func(*T) error) {
	By("Testing mutating the " + kind)
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

	By(fmt.Sprintf("Testing mutating the %s twice has the same result", kind))
	generated := *result
	err = mutate(result)
	Expect(err).NotTo(HaveOccurred())
	diff = cmp.Diff(generated, *result)
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
		{obj: &appsv1.Deployment{}, key: types.NamespacedName{Namespace: atom.Namespace, Name: getBareDeployment(atom).GetName()}},
		{obj: &corev1.ConfigMap{}, key: types.NamespacedName{Namespace: atom.Namespace, Name: configMapName}},
		{obj: &traefikiov1alpha1.Middleware{}, key: types.NamespacedName{Namespace: atom.Namespace, Name: getBareStripPrefixMiddleware(atom).GetName()}},
		{obj: &traefikiov1alpha1.Middleware{}, key: types.NamespacedName{Namespace: atom.Namespace, Name: getBareHeadersMiddleware(atom).GetName()}},
		{obj: &corev1.Service{}, key: types.NamespacedName{Namespace: atom.Namespace, Name: getBareService(atom).GetName()}},
		{obj: &traefikiov1alpha1.IngressRoute{}, key: types.NamespacedName{Namespace: atom.Namespace, Name: getBareIngressRoute(atom).GetName()}},
		{obj: &policyv1.PodDisruptionBudget{}, key: types.NamespacedName{Namespace: atom.Namespace, Name: getBarePodDisruptionBudget(atom).GetName()}},
	}
	for _, group := range getDownloadLinkGroups(atom.GetDownloadLinks()) {
		extraStruct := struct {
			obj client.Object
			key types.NamespacedName
		}{obj: &traefikiov1alpha1.Middleware{}, key: types.NamespacedName{Namespace: atom.Namespace, Name: getBareDownloadLinkMiddleware(atom, *group.index).GetName()}}

		structs = append(structs, extraStruct)
	}

	return structs
}

func Test_getGeneratorConfig(t *testing.T) {
	pdoknlv3.SetBlobEndpoint("http://localazurite.blob.azurite")
	type args struct {
		atom      *pdoknlv3.Atom
		ownerInfo *smoothoperatorv1.OwnerInfo
	}

	maxAtom, err := getAtom(testPath("maximum")+"input/atom.yaml", false)

	if err != nil {
		t.Errorf("getAtom() error = %v", err)
	}
	maxOwner, err := getOwnerInfo(testPath("maximum")+"input/ownerinfo.yaml", false)
	if err != nil {
		t.Errorf("getOwnerInfo() error = %v", err)
	}
	maxScenario := args{
		atom:      maxAtom,
		ownerInfo: maxOwner,
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
			name:       "maximum_scenario",
			args:       maxScenario,
			wantConfig: getTestGeneratorConfig(testPath("maximum") + "expected-output/configmap.yaml"),
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

func readTestFile(fileName string) (string, error) {
	dat, err := os.ReadFile(fileName)

	return string(dat), err
}

func getAtom(fileName string, ginkgo bool) (*pdoknlv3.Atom, error) {
	atom := &pdoknlv3.Atom{}
	data, err := os.ReadFile(fileName)
	if ginkgo {
		Expect(err).NotTo(HaveOccurred())
	}
	if err != nil {
		return nil, err
	}

	err = yaml.UnmarshalStrict(data, atom)
	if ginkgo {
		Expect(err).NotTo(HaveOccurred())
	}
	if err != nil {
		return nil, err
	}
	return atom, nil
}

func getOwnerInfo(fileName string, ginkgo bool) (*smoothoperatorv1.OwnerInfo, error) {
	owner := &smoothoperatorv1.OwnerInfo{}
	data, err := os.ReadFile(fileName)
	if ginkgo {
		Expect(err).NotTo(HaveOccurred())
	}
	if err != nil {
		return nil, err
	}
	err = yaml.UnmarshalStrict(data, owner)
	if ginkgo {
		Expect(err).NotTo(HaveOccurred())
	}
	if err != nil {
		return nil, err
	}
	return owner, nil
}

func getTestGeneratorConfig(fileName string) string {
	var configMap corev1.ConfigMap
	data, _ := os.ReadFile(fileName)
	_ = yaml.UnmarshalStrict(data, &configMap)
	return configMap.Data["values.yaml"]
}
