package clusteragent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/falcosecurity/falcosidekick/types"
	"github.com/kube-tarian/tarian/pkg/clusteragent/webhookserver"
	"github.com/kube-tarian/tarian/pkg/stringutil"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/scylladb/go-set/strset"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
)

const tarianRuleSpawnedProcess = "falco.tarian.dev/v1 detect spawned_process"

type FalcoSidekickListener struct {
	server *http.Server

	informers   informers.SharedInformerFactory
	configCache *ConfigCache

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient
	eventClient  tarianpb.EventClient

	actionHandler *actionHandler

	logger *logrus.Logger
}

func NewFalcoSidekickListener(
	logger *logrus.Logger,
	addr string,
	tarianServerAddress string,
	opts []grpc.DialOption,
	informers informers.SharedInformerFactory,
	configCache *ConfigCache,
	actionHandler *actionHandler) (*FalcoSidekickListener, error) {
	mux := http.NewServeMux()
	server := &http.Server{Addr: addr, Handler: mux}

	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)
	if err != nil {
		logger.WithError(err).Error("couldn't not connect to tarian-server")
		return nil, fmt.Errorf("NewFalcoSidekickListener: couldn't not connect to tarian-server: %w", err)
	}

	f := &FalcoSidekickListener{
		server:        server,
		informers:     informers,
		configCache:   configCache,
		grpcConn:      grpcConn,
		configClient:  tarianpb.NewConfigClient(grpcConn),
		eventClient:   tarianpb.NewEventClient(grpcConn),
		actionHandler: actionHandler,
	}

	mux.HandleFunc("/", f.handleFalcoAlert)

	return f, nil
}

func (f *FalcoSidekickListener) handleFalcoAlert(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "Request body can not be empty", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST http method is supported", http.StatusBadRequest)
		return
	}

	body, _ := io.ReadAll(r.Body)
	falcopayload, err := newFalcoPayload(bytes.NewBuffer(body))

	if err != nil {
		f.logger.WithError(err).Error("error while decoding falco payload")
		http.Error(w, "Error encountered while decoding falco payload", http.StatusBadRequest)
		return
	}

	err = f.processFalcoPayload(&falcopayload)
	if err != nil {
		f.logger.WithError(err).Error("error while processing falco payload")
	}

	w.WriteHeader(200)
	_, err = w.Write([]byte("OK"))
	if err != nil {
		f.logger.WithError(err).Error("error while writing response")
	}
}

// sanitizeK8sResourceName sanitizes input from falco for additional security and satisfies codeql analysis:
// "This log write receives unsanitized user input from"
func sanitizeK8sResourceName(str string) string {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names
	// RFC 1123 Label Names, contain no more than 253 characters
	return strings.Replace(stringutil.Truncate(str, 253), "\n", "", -1)
}

func (f *FalcoSidekickListener) processFalcoPayload(payload *types.FalcoPayload) error {
	if payload.Rule == tarianRuleSpawnedProcess {
		if !f.configCache.IsConstraintInitialized() {
			f.logger.WithField("payload", payload).
				Info("can not validate process because constraint is not yet initialized")
			return nil
		}

		outputFields := payload.OutputFields
		k8sPodName := fmt.Sprintf("%s", outputFields["k8s.pod.name"])
		k8sNsName := fmt.Sprintf("%s", outputFields["k8s.ns.name"])
		procName := fmt.Sprintf("%s", outputFields["proc.name"])

		k8sPod, err := f.informers.Core().V1().Pods().Lister().Pods(k8sNsName).Get(k8sPodName)

		if err != nil {
			f.logger.WithFields(logrus.Fields{
				"pod_name":  sanitizeK8sResourceName(k8sPodName),
				"namespace": sanitizeK8sResourceName(k8sNsName),
				"error":     err,
			}).Error("error while getting pod by name and namespace")
			return nil
		}

		_, threatScanAnnotationPresent := k8sPod.Annotations[webhookserver.ThreatScanAnnotation]
		registerAnnotationValue, registerAnnotationPresent := k8sPod.Annotations[webhookserver.RegisterAnnotation]
		if !threatScanAnnotationPresent && !registerAnnotationPresent {
			return nil
		}

		registerProcess := false
		registerRules := strings.Split(registerAnnotationValue, ",")
		for _, rule := range registerRules {
			switch strings.TrimSpace(rule) {
			case "processes":
				registerProcess = true
			case "all":
				registerProcess = true
			}
		}

		matchedConstraints := f.getConstraintsMatchingLabels(k8sPod.GetLabels())

		// if there are existing constraints, and this process matches it, just return
		// because no action would be done for register and threat-scan mode
		if len(matchedConstraints) > 0 && f.validateProcessAgainstConstraints(procName, matchedConstraints) {
			return nil
		}

		if registerProcess {
			f.registerConstraintFromTarianRuleSpawnedProcessAlert(payload, k8sPod)
		} else {
			if len(matchedConstraints) == 0 {
				return nil
			}

			event, err := f.ingestEventFromTarianRuleSpawnedProcessAlert(payload, k8sPod)
			if err == nil {
				f.actionHandler.QueueEvent(event)
			}
		}
	} else {
		outputFields := payload.OutputFields
		k8sPodName := fmt.Sprintf("%s", outputFields["k8s.pod.name"])
		k8sNsName := fmt.Sprintf("%s", outputFields["k8s.ns.name"])

		k8sPod, err := f.informers.Core().V1().Pods().Lister().Pods(k8sNsName).Get(k8sPodName)

		if err != nil {
			f.logger.WithFields(logrus.Fields{
				"pod_name":  sanitizeK8sResourceName(k8sPodName),
				"namespace": sanitizeK8sResourceName(k8sNsName),
				"error":     err,
			}).Error("error while getting pod by name and namespace")
			return nil
		}

		event, err := f.ingestEventFromGenericFalcoAlert(payload, k8sPod)
		if err == nil {
			f.actionHandler.QueueEvent(event)
		}
	}

	return nil
}

func newFalcoPayload(payload io.Reader) (types.FalcoPayload, error) {
	var falcopayload types.FalcoPayload

	d := json.NewDecoder(payload)
	d.UseNumber()

	err := d.Decode(&falcopayload)
	if err != nil {
		return types.FalcoPayload{}, fmt.Errorf("newFalcoPayload: %w", err)
	}

	return falcopayload, nil
}

func (f *FalcoSidekickListener) getConstraintsMatchingLabels(labels map[string]string) []*tarianpb.Constraint {
	matchedConstraints := []*tarianpb.Constraint{}

	k8sPodLabelSet := strset.New()
	for k, v := range labels {
		k8sPodLabelSet.Add(k + "=" + v)
	}

	for _, constraint := range f.configCache.GetConstraints() {
		if constraint.GetSelector() == nil || constraint.GetSelector().GetMatchLabels() == nil {
			continue
		}

		constraintSelectorLabelSet := strset.New()
		for _, l := range constraint.GetSelector().GetMatchLabels() {
			constraintSelectorLabelSet.Add(l.GetKey() + "=" + l.GetValue())
		}

		if k8sPodLabelSet.IsSubset(constraintSelectorLabelSet) {
			matchedConstraints = append(matchedConstraints, constraint)
		}
	}

	return matchedConstraints
}

func (f *FalcoSidekickListener) validateProcessAgainstConstraints(processName string, constraints []*tarianpb.Constraint) bool {
	for _, constraint := range constraints {
		if constraint.GetAllowedProcesses() == nil {
			continue
		}

		for _, allowedProcess := range constraint.GetAllowedProcesses() {
			if allowedProcess.GetRegex() == "" {
				continue
			}

			rgx, err := regexp.Compile(allowedProcess.GetRegex())

			if err != nil {
				f.logger.WithError(err).Error("can not compile regex")
				continue
			}

			f.logger.WithField("expr", rgx.String()).Debug("checking process name against regex")

			if rgx.MatchString(processName) {
				return true
			}
		}
	}

	return false
}

func (f *FalcoSidekickListener) registerConstraintFromTarianRuleSpawnedProcessAlert(payload *types.FalcoPayload, pod *v1.Pod) {
	outputFields := payload.OutputFields
	k8sPodName := fmt.Sprintf("%s", outputFields["k8s.pod.name"])
	k8sNsName := fmt.Sprintf("%s", outputFields["k8s.ns.name"])

	procName := fmt.Sprintf("%s", outputFields["proc.name"])
	allowedProcessRules := []*tarianpb.AllowedProcessRule{{Regex: &procName}}

	podLabels := pod.GetLabels()
	delete(podLabels, "pod-template-hash")

	req := &tarianpb.AddConstraintRequest{
		Constraint: &tarianpb.Constraint{
			Kind:      tarianpb.KindConstraint,
			Namespace: k8sNsName,
			Name:      k8sPodName + "-" + strconv.FormatInt(time.Now().UnixNano()/time.Hour.Milliseconds(), 10),
			Selector: &tarianpb.Selector{
				MatchLabels: matchLabelsFromPodLabels(podLabels),
			},
			AllowedProcesses: allowedProcessRules,
		},
	}

	response, err := f.configClient.AddConstraint(context.Background(), req)

	if err != nil {
		f.logger.WithError(err).Error("error while registering process constraint")
	} else {
		f.logger.WithField("response", response).Info("registered process constraint")
	}
}

func (f *FalcoSidekickListener) ingestEventFromTarianRuleSpawnedProcessAlert(payload *types.FalcoPayload, pod *v1.Pod) (*tarianpb.Event, error) {
	outputFields := payload.OutputFields
	k8sPodName := fmt.Sprintf("%s", outputFields["k8s.pod.name"])
	k8sNsName := fmt.Sprintf("%s", outputFields["k8s.ns.name"])

	var pbPod *tarianpb.Pod
	labels := []*tarianpb.Label{}
	for k, v := range pod.GetLabels() {
		labels = append(labels, &tarianpb.Label{Key: k, Value: v})
	}

	if outputFields != nil {
		pbPod = &tarianpb.Pod{
			Uid:       string(pod.GetUID()),
			Name:      k8sPodName,
			Namespace: k8sNsName,
			Labels:    labels,
		}
	}

	violatedProcesses := make([]*tarianpb.Process, 1)
	pid, err := strconv.ParseInt(fmt.Sprintf("%v", outputFields["proc.pid"]), 10, 32)
	if err != nil {
		procPid := fmt.Sprintf("%s", outputFields["proc.pid"])
		procPid = strings.Replace(procPid, "\n", "", -1)
		f.logger.WithFields(logrus.Fields{
			"proc.pid": procPid,
			"err":      err,
		}).Warn("expected proc.pid to be int value, but got non-int")
	}
	violatedProcesses[0] = &tarianpb.Process{Pid: int32(pid), Name: fmt.Sprintf("%s", outputFields["proc.name"])}

	event := &tarianpb.Event{
		Type:            tarianpb.EventTypeFalcoAlert,
		ClientTimestamp: timestamppb.New(payload.Time),
		Targets: []*tarianpb.Target{
			{
				Pod:               pbPod,
				ViolatedProcesses: violatedProcesses,
			},
		},
	}

	req := &tarianpb.IngestEventRequest{
		Event: event,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	response, err := f.eventClient.IngestEvent(ctx, req)
	defer cancel()

	if err != nil {
		f.logger.WithError(err).Error("error while reporting falco spawned process")
	} else {
		f.logger.WithField("response", response).Info("ingest event response")
	}

	return event, err
}

func (f *FalcoSidekickListener) ingestEventFromGenericFalcoAlert(payload *types.FalcoPayload, k8sPod *v1.Pod) (*tarianpb.Event, error) {
	outputFields := payload.OutputFields
	var pod *tarianpb.Pod
	if outputFields != nil {
		labels := []*tarianpb.Label{}
		for k, v := range k8sPod.GetLabels() {
			labels = append(labels, &tarianpb.Label{Key: k, Value: v})
		}

		pod = &tarianpb.Pod{
			Name:      fmt.Sprintf("%s", outputFields["k8s.pod.name"]),
			Namespace: fmt.Sprintf("%s", outputFields["k8s.ns.name"]),
			Labels:    labels,
		}
	}

	outputFieldsStr := map[string]string{}
	for k, v := range payload.OutputFields {
		outputFieldsStr[k] = fmt.Sprintf("%v", v)
	}

	event := &tarianpb.Event{
		Type:            tarianpb.EventTypeFalcoAlert,
		ClientTimestamp: timestamppb.New(payload.Time),
		Targets: []*tarianpb.Target{
			{
				Pod: pod,
				FalcoAlert: &tarianpb.FalcoAlert{
					Rule:         payload.Rule,
					Priority:     tarianpb.FalcoPriorityFromString(payload.Priority.String()),
					Output:       payload.Output,
					OutputFields: outputFieldsStr,
				},
			},
		},
	}

	req := &tarianpb.IngestEventRequest{
		Event: event,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	response, err := f.eventClient.IngestEvent(ctx, req)
	defer cancel()

	if err != nil {
		f.logger.WithError(err).Error("error while reporting falco alerts")
	} else {
		f.logger.WithField("response", response).Debug("ingest event response")
	}

	return event, fmt.Errorf("ingestEventFromGenericFalcoAlert: %w", err)
}

func matchLabelsFromPodLabels(labels map[string]string) []*tarianpb.MatchLabel {
	matchLabels := make([]*tarianpb.MatchLabel, len(labels))

	i := 0
	for k, v := range labels {
		matchLabels[i] = &tarianpb.MatchLabel{Key: k, Value: v}
		i++
	}

	return matchLabels
}
