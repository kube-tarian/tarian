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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

// NewManager creates a new controller manager for managing webhooks and controllers.
// Parameters:
//   - logger: The logger instance for logging.
//   - port: The port to bind the manager's webhook server.
//   - healthProbeBindAddress: The address for health probes.
//   - leaderElection: A flag indicating whether leader election should be enabled.
//
// Returns:
//   - manager.Manager: The created controller manager.
//   - error: An error, if any, encountered during manager creation.
func NewManager(logger *logrus.Logger, port int, healthProbeBindAddress string, leaderElection bool) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
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

// RegisterControllers registers controllers for webhook operations.
// Parameters:
//   - logger: The logger instance for logging.
//   - mgr: The controller manager.
//   - cfg: Configuration for the PodAgent container.
func RegisterControllers(logger *logrus.Logger, mgr manager.Manager, cfg PodAgentContainerConfig) {
	mgr.GetWebhookServer().Register(
		"/inject-pod-agent",
		&webhook.Admission{
			Handler: &PodAgentInjector{
				Client:  mgr.GetClient(),
				decoder: admission.NewDecoder(scheme),
				config:  cfg,
				logger:  logger,
			},
		},
	)
}

// RegisterCertRotator registers the certificate rotator for managing certificates used in webhooks.
// Parameters:
//   - logger: The logger instance for logging.
//   - mgr: The controller manager.
//   - isReady: A channel indicating readiness.
//   - namespace: The Kubernetes namespace.
//   - mutatingWebhookConfigurationName: The name of the mutating webhook configuration.
//   - secretName: The name of the Kubernetes secret containing certificates.
//
// Returns:
//   - error: An error, if any, encountered during cert rotator registration.
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
