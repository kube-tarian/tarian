package server

import (
	"context"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/devopstoday11/tarian/pkg/store"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"

	clientruntime "github.com/go-openapi/runtime/client"
)

const (
	defaultAmApiv2path = "/api/v2"
)

type AlertDispatcher struct {
	amClient                *client.Alertmanager
	alertEvaluationInterval time.Duration
}

func NewAlertDispatcher(amURL *url.URL, alertEvaluationInterval time.Duration) *AlertDispatcher {
	amClient := NewAlertmanagerClient(amURL)

	return &AlertDispatcher{
		amClient, alertEvaluationInterval,
	}
}

func NewAlertmanagerClient(amURL *url.URL) *client.Alertmanager {
	cr := clientruntime.New(amURL.Host, path.Join(amURL.Path, defaultAmApiv2path), []string{amURL.Scheme})

	if amURL.User != nil {
		password, _ := amURL.User.Password()
		cr.DefaultAuthentication = clientruntime.BasicAuth(amURL.User.Username(), password)
	}

	return client.New(cr, strfmt.Default)
}

func (a *AlertDispatcher) LoopSendAlerts(ctx context.Context, es store.EventStore) {
	for {
		events, err := es.FindWhereAlertNotSent()

		if err != nil {
			logger.Errorw("alertdispatcher: error while finding events to alert", "err", err)
		}

		for _, event := range events {
			if event.GetType() == "violation" {
				err := a.SendAlert(event)

				if err == nil {
					es.UpdateAlertSent(event.GetUid())
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
		labels["violating_processes"] = violatingProcessesToString(target.GetViolatingProcesses())

		pa := &models.PostableAlert{Alert: models.Alert{Labels: labels}}

		alertParams := alert.NewPostAlertsParamsWithContext(ctx)
		alertParams.Alerts = append(alertParams.Alerts, pa)

		status, err := a.amClient.Alert.PostAlerts(alertParams)

		if err != nil {
			logger.Error("error while sending alerts", "err", err)
		} else {
			logger.Info("alerts sent to alertmanager", "result", status.Error())
		}

		return err
	}
}

func violatingProcessesToString(processes []*tarianpb.Process) string {
	str := strings.Builder{}

	for i, p := range processes {
		str.WriteString(strconv.Itoa(int(p.GetId())))
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
