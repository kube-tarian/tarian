package webhookserver

import (
	"fmt"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"github.com/sirupsen/logrus"
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
	scheme = runtime.NewScheme()
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

func NewManager(logger *logrus.Logger, port int, healthProbeBindAddress string, leaderElection bool) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     "0",
		Port:                   port,
		HealthProbeBindAddress: healthProbeBindAddress,
		LeaderElection:         leaderElection,
		LeaderElectionID:       leaderElectionID,
	})
	if err != nil {
		logger.WithError(err).Error("unable to start manager")
		return nil, fmt.Errorf("NewManager: unable to start manager: %w", err)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.WithError(err).Error("unable to set up health check")
		return nil, fmt.Errorf("NewManager: unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.WithError(err).Error("unable to set up ready check")
		return nil, fmt.Errorf("NewManager: unable to set up ready check: %w", err)
	}

	return mgr, nil

}

func RegisterControllers(logger *logrus.Logger, mgr manager.Manager, cfg PodAgentContainerConfig) {
	mgr.GetWebhookServer().Register(
		"/inject-pod-agent",
		&webhook.Admission{
			Handler: &PodAgentInjector{
				Client: mgr.GetClient(),
				config: cfg,
				logger: logger,
			},
		},
	)
}

func RegisterCertRotator(logger *logrus.Logger, mgr manager.Manager, isReady chan struct{},
	namespace string, mutatingWebhookConfigurationName string, secretName string) error {
	dnsName := "*." + namespace + ".svc"
	certDir := "/tmp/k8s-webhook-server/serving-certs"

	var webhooks = []rotator.WebhookInfo{
		{
			Name: mutatingWebhookConfigurationName,
			Type: rotator.Mutating,
		},
	}

	logger.Info("setting up cert rotation")
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
		logger.WithError(err).Error("unable to set up cert rotation")
		return fmt.Errorf("RegisterCertRotator: unable to set up cert rotation: %w", err)
	}
	return nil
}
