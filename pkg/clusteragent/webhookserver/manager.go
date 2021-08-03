package webhookserver

import (
	"os"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	managedSecretName = "tarian-webhook-server-cert"
	serviceName       = "tarian-controller-manager"
	caName            = "tarian-ca"
	caOrganization    = "tarian"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func NewManager() manager.Manager {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     "0",
		Port:                   9443,    //  TODO: extract to CLI flag
		HealthProbeBindAddress: ":8081", // TODO: extract to CLI flag
		LeaderElection:         false,   // TODO: extract to CLI flag
		LeaderElectionID:       "0f4c7cb2.k8s.tarian.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	return mgr

}

func RegisterControllers(mgr manager.Manager, cfg PodAgentContainerConfig) {
	mgr.GetWebhookServer().Register(
		"/inject-pod-agent",
		&webhook.Admission{
			Handler: &PodAgentInjector{
				Client: mgr.GetClient(),
				config: cfg,
			},
		},
	)
}

func RegisterCertRotator(mgr manager.Manager, isReady chan struct{}) {
	dnsName := "*.tarian-system.svc"
	certDir := "/tmp/k8s-webhook-server/serving-certs"

	var webhooks = []rotator.WebhookInfo{
		{
			Name: "tarian-mutating-webhook-configuration",
			Type: rotator.Mutating,
		},
	}

	setupLog.Info("setting up cert rotation")
	if err := rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: "tarian-system", // TODO: extract
			Name:      managedSecretName,
		},
		CertDir:        certDir,
		CAName:         caName,
		CAOrganization: caOrganization,
		DNSName:        dnsName,
		IsReady:        isReady,
		Webhooks:       webhooks,
	}); err != nil {
		setupLog.Error(err, "unable to set up cert rotation")
		os.Exit(1)
	}
}
