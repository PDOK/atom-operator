/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/zapr"
	"github.com/pdok/smooth-operator/pkg/integrations/logging"
	"github.com/peterbourgon/ff"
	"go.uber.org/zap/zapcore"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	traefikiov1alpha1 "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	"github.com/pdok/atom-operator/internal/controller"
	webhookpdoknlv3 "github.com/pdok/atom-operator/internal/webhook/v3"
	// +kubebuilder:scaffold:imports
)

const (
	defaultAtomGeneratorImage = "acrpdokprodman.azurecr.io/mirror/docker.io/pdok/atom-generator:0.6.2"
	defaultLighttpdImage      = "acrpdokprodman.azurecr.io/mirror/docker.io/pdok/lighttpd:1.4.67"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(pdoknlv3.AddToScheme(scheme))
	utilruntime.Must(traefikiov1alpha1.AddToScheme(scheme))
	utilruntime.Must(smoothoperatorv1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

//nolint:funlen
func main() {
	var metricsAddr string
	var certDir string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var baseURL string
	var blobEndpoint string
	var atomGeneratorImage string
	var lighttpdImage string
	var tlsOpts []func(*tls.Config)
	var slackWebhookURL string
	var logLevel int
	var csp string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&certDir, "cert-dir", "", "CertDir contains the webhook server key and certificate. Defaults to <temp-dir>/k8s-webhook-server/serving-certs.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.StringVar(&baseURL, "atom-baseurl", "", "The base url which is used in the atom service.")
	flag.StringVar(&blobEndpoint, "blob-endpoint", "", "The blobstore endpoint used for file downloads.")
	flag.StringVar(&atomGeneratorImage, "atom-generator-image", defaultAtomGeneratorImage, "The image to use in the Atom generator init-container.")
	flag.StringVar(&lighttpdImage, "lighttpd-image", defaultLighttpdImage, "The image to use in the Atom pod.")
	flag.StringVar(&slackWebhookURL, "slack-webhook-url", "", "The webhook url for sending slack messages. Disabled if left empty")
	flag.IntVar(&logLevel, "log-level", 0, "The zapcore loglevel. 0 = info, 1 = warn, 2 = error")
	flag.StringVar(&csp, "csp", "", "Content-Security-Policy to serve as a HTTP header")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)

	if err := ff.Parse(flag.CommandLine, os.Args[1:], ff.WithEnvVarNoPrefix()); err != nil {
		setupLog.Error(err, "unable to parse flags")
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	//nolint:gosec
	levelEnabler := zapcore.Level(logLevel)
	zapLogger, _ := logging.SetupLogger("atom-operator", slackWebhookURL, levelEnabler)
	logrLogger := zapr.NewLogger(zapLogger)

	ctrl.SetLogger(logrLogger)

	if baseURL == "" {
		setupLog.Error(errors.New("baseURL is required"), "A value for baseURL must be specified.")
		os.Exit(1)
	}
	pdoknlv3.SetBaseURL(baseURL)

	if blobEndpoint == "" {
		setupLog.Error(errors.New("blobEndpoint is required"), "A value for blobEndpoint must be specified.")
		os.Exit(1)
	}
	pdoknlv3.SetBlobEndpoint(blobEndpoint)

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		CertDir: certDir,
		TLSOpts: tlsOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "29e70f77.pdok.nl",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.AtomReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		AtomGeneratorImage: atomGeneratorImage,
		LighttpdImage:      lighttpdImage,
		CSP:                csp,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Atom")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {

		if err = webhookpdoknlv3.SetupAtomWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Atom")
			os.Exit(1)
		}

	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
