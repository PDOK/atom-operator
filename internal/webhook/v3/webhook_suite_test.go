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
	"crypto/tls"
	"errors"
	"fmt"
	samples "github.com/pdok/atom-operator/internal/webhook/v3/ownerinfo-test"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	"golang.org/x/tools/go/packages"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // ginkgo bdd
	. "github.com/onsi/gomega"    //nolint:revive // ginkgo bdd

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx       context.Context
	cancel    context.CancelFunc
	k8sClient client.Client
	cfg       *rest.Config
	testEnv   *envtest.Environment
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	//nolint:fatcontext
	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	scheme := runtime.NewScheme()
	err = pdoknlv3.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = smoothoperatorv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	By("bootstrapping test environment")
	traefikCRDPath := must(getTraefikCRDPath())
	ownerInfoCRDPath := must(getOwnerInfoCRDPath())
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: false,

		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "config", "webhook")},
		},
		CRDInstallOptions: envtest.CRDInstallOptions{
			Scheme: nil,
			Paths: []string{
				filepath.Join("..", "..", "..", "config", "crd", "bases", "pdok.nl_atoms.yaml"),
				traefikCRDPath,
				ownerInfoCRDPath,
			},
			ErrorIfPathMissing: true,
		},
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("creating manager namespace")

	clientset, err := kubernetes.NewForConfig(cfg)
	namespace22 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "services",
		},
	}
	clientset.CoreV1().Namespaces().Create(ctx, namespace22, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")
	ownerInfo, err := samples.OwnerInfoSample()
	Expect(err).NotTo(HaveOccurred())
	Expect(ownerInfo).NotTo(BeNil())

	err = k8sClient.Create(ctx, ownerInfo)
	Expect(err).NotTo(HaveOccurred())

	// start webhook server using Manager.
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	err = SetupAtomWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:webhook

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready.
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}

		return conn.Close()
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func getOwnerInfoCRDPath() (string, error) {
	smoothOperatorModule, err := getModule("github.com/pdok/smooth-operator")
	if err != nil {
		return "", err
	}
	if smoothOperatorModule.Dir == "" {
		return "", errors.New("cannot find path for smooth-operator module")
	}
	return filepath.Join(smoothOperatorModule.Dir, "config", "crd", "bases", "pdok.nl_ownerinfo.yaml"), nil
}

func getTraefikCRDPath() (string, error) {
	traefikModule, err := getModule("github.com/traefik/traefik/v3")
	if err != nil {
		return "", err
	}
	if traefikModule.Dir == "" {
		return "", errors.New("cannot find path for traefik module")
	}
	return filepath.Join(traefikModule.Dir, "integration", "fixtures", "k8s", "01-traefik-crd.yml"), nil
}

func getModule(name string) (module *packages.Module, err error) {
	out, err := exec.Command("go", "list", "-json", "-m", name).Output()
	if err != nil {
		return
	}
	module = &packages.Module{}
	err = json.Unmarshal(out, module)
	return
}
