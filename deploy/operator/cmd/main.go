package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/ggid/ggid/deploy/operator/internal/api"
	iamv1alpha1 "github.com/ggid/ggid/deploy/operator/api/v1alpha1"
	"github.com/ggid/ggid/deploy/operator/internal/controller"
	"github.com/ggid/ggid/deploy/operator/internal/provisioning"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(iamv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var gatewayURL string
	var apiAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Metrics server bind address")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Probe server bind address")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election")
	flag.StringVar(&gatewayURL, "gateway-url", os.Getenv("GGID_GATEWAY_URL"), "GGID gateway URL for shared-mode tenant provisioning")
	flag.StringVar(&apiAddr, "api-bind-address", ":9090", "API server bind address for provisioning endpoints")
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ggid-operator.iam.ggid.dev",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Tenant provisioner (shared mode)
	tenantProvisioner := provisioning.NewTenantProvisioner(gatewayURL)

	// Instance provisioner (dedicated mode)
	instanceProvisioner := provisioning.NewInstanceProvisioner()

	if err := (&controller.GGIDTenantReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Provisioner:   tenantProvisioner,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create GGIDTenant controller")
		os.Exit(1)
	}

	if err := (&controller.GGIDInstanceReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Provisioner: instanceProvisioner,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create GGIDInstance controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start the provisioning API server in a goroutine.
	// It uses the manager's cached client for CR operations.
	apiServer := api.NewAPIServer(mgr.GetClient(), scheme, gatewayURL)
	go func() {
		if err := apiServer.Start(apiAddr); err != nil {
			setupLog.Error(err, "API server stopped")
		}
	}()

	setupLog.Info("starting GGID operator", "gateway-url", gatewayURL, "api-addr", apiAddr)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
