package server

import (
	"context"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/sirupsen/logrus"

	clientruntime "github.com/go-openapi/runtime/client"
)

const (
	defaultAmApiv2path = "/api/v2"
)

// AlertDispatcher is responsible for dispatching alerts to Alertmanager.
type AlertDispatcher struct {
	amClient                *client.Alertmanager
	alertEvaluationInterval time.Duration

	logger *logrus.Logger
}

// NewAlertDispatcher creates a new AlertDispatcher instance.
//
// Parameters:
// - logger: The logger to use for logging.
// - amURL: The URL of the Alertmanager.
// - alertEvaluationInterval: The interval for evaluating and sending alerts.
//
// Returns:
// - *AlertDispatcher: A new instance of AlertDispatcher.
func NewAlertDispatcher(logger *logrus.Logger, amURL *url.URL, alertEvaluationInterval time.Duration) *AlertDispatcher {
	amClient := NewAlertmanagerClient(amURL)

	return &AlertDispatcher{
		amClient:                amClient,
		alertEvaluationInterval: alertEvaluationInterval,
		logger:                  logger,
	}
}

// NewAlertmanagerClient creates a new Alertmanager client.
//
// Parameters:
// - amURL: The URL of the Alertmanager.
//
// Returns:
// - *client.Alertmanager: A new Alertmanager client.
func NewAlertmanagerClient(amURL *url.URL) *client.Alertmanager {
	cr := clientruntime.New(amURL.Host, path.Join(amURL.Path, defaultAmApiv2path), []string{amURL.Scheme})

	if amURL.User != nil {
		password, _ := amURL.User.Password()
		cr.DefaultAuthentication = clientruntime.BasicAuth(amURL.User.Username(), password)
	}

	return client.New(cr, strfmt.Default)
}

// LoopSendAlerts continuously sends alerts to Alertmanager based on events from the EventStore.
//
// Parameters:
// - ctx: The context for the operation.
// - es: The EventStore to retrieve events from.
func (a *AlertDispatcher) LoopSendAlerts(ctx context.Context, es store.EventStore) {
	for {
		events, err := es.FindWhereAlertNotSent()
		if err != nil {
			a.logger.WithError(err).Error("alertdispatcher: error while finding events to alert")
		}
		for _, event := range events {
			if event.GetType() == tarianpb.EventTypeViolation || event.GetType() == tarianpb.EventTypeFalcoAlert {
				err := a.SendAlert(event)

				if err == nil {
					err = es.UpdateAlertSent(event.GetUid())
					if err != nil {
						a.logger.WithError(err).Warn("alertdispatcher: error while updating alert sent")
					}
					a.logger.Debug("alertdispatcher: AlertSentAt time upated successfully", event.GetUid())
				}
			}
		}

		select {
		case <-time.After(a.alertEvaluationInterval):
		case <-ctx.Done():
			return
		}
	}
}

// SendAlert sends an alert to Alertmanager based on the provided event.
//
// Parameters:
// - event: The event to generate an alert from.
//
// Returns:
// - error: An error, if any, during the alert sending process.
func (a *AlertDispatcher) SendAlert(event *tarianpb.Event) error {
	for _, target := range event.GetTargets() {
		if target.GetPod() == nil {
			continue
		}

		ctx := context.Background()

		labels := make(models.LabelSet)
		labels["type"] = event.GetType()
		labels["serverTimestamp"] = event.GetServerTimestamp().AsTime().Format(time.RFC3339)
		labels["pod_namespace"] = target.GetPod().GetNamespace()
		labels["pod_name"] = target.GetPod().GetName()
		labels["pod_uid"] = target.GetPod().GetUid()

		if target.GetViolatedProcesses() != nil {
			labels["violated_processes"] = violatedProcessesToString(target.GetViolatedProcesses())
		}

		if target.GetViolatedFiles() != nil {
			labels["violated_files"] = violatedFilesToString(target.GetViolatedFiles())
		}

		if target.GetFalcoAlert() != nil {
			labels["falco_alert"] = target.GetFalcoAlert().GetOutput()
		}

		pa := &models.PostableAlert{Alert: models.Alert{Labels: labels}}

		alertParams := alert.NewPostAlertsParamsWithContext(ctx)
		alertParams.Alerts = append(alertParams.Alerts, pa)

		status, err := a.amClient.Alert.PostAlerts(alertParams)

		if err != nil {
			a.logger.Error("error while sending alerts: ", err)
		} else {
			a.logger.WithField("result", status.Error()).Info("alerts sent to alertmanager")
		}

		return err
	}

	return nil
}

func violatedProcessesToString(processes []*tarianpb.Process) string {
	str := strings.Builder{}

	for i, p := range processes {
		str.WriteString(strconv.Itoa(int(p.GetPid())))
		str.WriteString(":")
		str.WriteString(p.GetName())

		if i < len(processes)-1 {
			str.WriteString(", ")
		}

		if i >= 10 {
			str.WriteString("... ")
			str.WriteString(strconv.Itoa(int(len(processes) - i - 1)))
			str.WriteString(" more")
			break
		}
	}

	return str.String()
}

func violatedFilesToString(violatedFiles []*tarianpb.ViolatedFile) string {
	str := strings.Builder{}

	for i, f := range violatedFiles {
		str.WriteString(f.GetName())

		if i < len(violatedFiles)-1 {
			str.WriteString(", ")
		}

		if i >= 10 {
			str.WriteString("... ")
			str.WriteString(strconv.Itoa(int(len(violatedFiles) - i - 1)))
			str.WriteString(" more")
			break
		}
	}

	return str.String()
}
