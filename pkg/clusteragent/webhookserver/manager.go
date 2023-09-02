package webhookserver

import (
	"os"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	caName         = "tarian-ca"
	caOrganization = "tarian"

	leaderElectionID = "0f4c7cb2.k8s.tarian.dev"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func NewManager(port int, healthProbeBindAddress string, leaderElection bool) manager.Manager {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     "0",
		Port:                   port,
		HealthProbeBindAddress: healthProbeBindAddress,
		LeaderElection:         leaderElection,
		LeaderElectionID:       leaderElectionID,
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

func RegisterControllers(mgr manager.Manager, cfg PodAgentContainerConfig, logger *zap.SugaredLogger) {
	mgr.GetWebhookServer().Register(
		"/inject-pod-agent",
		&webhook.Admission{
			Handler: &PodAgentInjector{
				Client:  mgr.GetClient(),
				decoder: admission.NewDecoder(mgr.GetScheme()),
				config:  cfg,
				logger:  logger,
			},
		},
	)
}

func RegisterCertRotator(mgr manager.Manager, isReady chan struct{}, namespace string, mutatingWebhookConfigurationName string, secretName string) {
	dnsName := "*." + namespace + ".svc"
	certDir := "/tmp/k8s-webhook-server/serving-certs"

	var webhooks = []rotator.WebhookInfo{
		{
			Name: mutatingWebhookConfigurationName,
			Type: rotator.Mutating,
		},
	}

	setupLog.Info("setting up cert rotation")
	if err := rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: namespace,
			Name:      secretName,
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
