package clusteragent

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/falcosecurity/client-go/pkg/api/outputs"
	"github.com/falcosecurity/client-go/pkg/client"
	"github.com/kube-tarian/tarian/pkg/clusteragent/webhookserver"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/scylladb/go-set/strset"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

const tarianRuleSpawnedProcess = "falco.tarian.dev/v1 detect spawned_process"

type FalcoAlertsSubscriber struct {
	client       *client.Client
	grpcConn     *grpc.ClientConn
	eventClient  tarianpb.EventClient
	configClient tarianpb.ConfigClient
	cancelCtx    context.Context
	cancelFunc   context.CancelFunc

	kubeClientset *kubernetes.Clientset
	informers     informers.SharedInformerFactory
	configCache   *ConfigCache

	actionHandler *actionHandler
}

func NewFalcoAlertsSubscriber(
	tarianServerAddress string,
	opts []grpc.DialOption,
	config *client.Config,
	actionHandler *actionHandler,
	kubeClientset *kubernetes.Clientset,
	informers informers.SharedInformerFactory,
	configCache *ConfigCache) (*FalcoAlertsSubscriber, error) {
	falcoClient, err := client.NewForConfig(context.Background(), config)

	if err != nil {
		return nil, err
	}

	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)
	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &FalcoAlertsSubscriber{
		client:        falcoClient,
		grpcConn:      grpcConn,
		eventClient:   tarianpb.NewEventClient(grpcConn),
		configClient:  tarianpb.NewConfigClient(grpcConn),
		cancelCtx:     ctx,
		cancelFunc:    cancel,
		actionHandler: actionHandler,
		kubeClientset: kubeClientset,
		informers:     informers,
		configCache:   configCache,
	}, nil
}

func (f *FalcoAlertsSubscriber) Start() {
	for {
		select {
		case <-time.After(time.Second):
			err := f.client.OutputsWatch(f.cancelCtx, f.ProcessFalcoOutput, time.Second*1)
			if err != nil {
				logger.Error(err)
			}
		case <-f.cancelCtx.Done():
			return
		}
	}
}

func (f *FalcoAlertsSubscriber) Close() {
	f.cancelFunc()
	f.client.Close()
	f.grpcConn.Close()
}

func (f *FalcoAlertsSubscriber) ProcessFalcoOutput(res *outputs.Response) error {
	if res.GetRule() == tarianRuleSpawnedProcess {
		if !f.configCache.IsConstraintInitialized() {
			logger.Infow("can not validate process because constraint is not yet initialized")
			return nil
		}

		outputFields := res.GetOutputFields()
		k8sPodName := outputFields["k8s.pod.name"]
		k8sNsName := outputFields["k8s.ns.name"]
		procName := outputFields["proc.name"]

		k8sPod, err := f.informers.Core().V1().Pods().Lister().Pods(k8sNsName).Get(k8sPodName)

		if err != nil {
			logger.Errorw("error while getting pod by name and namespace", "err", err, "name", k8sPodName, "namespace", k8sNsName)
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
			f.registerConstraintFromTarianRuleSpawnedProcessAlert(res, k8sPod)
		} else {
			if len(matchedConstraints) == 0 {
				return nil
			}

			event, err := f.ingestEventFromTarianRuleSpawnedProcessAlert(res, k8sPod)
			if err == nil {
				f.actionHandler.QueueEvent(event)
			}
		}
	} else {
		event, err := f.ingestEventFromGenericFalcoAlert(res)
		if err == nil {
			f.actionHandler.QueueEvent(event)
		}
	}

	return nil
}

func (f *FalcoAlertsSubscriber) registerConstraintFromTarianRuleSpawnedProcessAlert(res *outputs.Response, pod *v1.Pod) {
	outputFields := res.GetOutputFields()
	k8sPodName := outputFields["k8s.pod.name"]
	k8sNsName := outputFields["k8s.ns.name"]

	procName := outputFields["proc.name"]
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
		logger.Errorw("error while registering process constraint", "err", err)
	} else {
		logger.Debugw("add constraint response", "response", response)
	}
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

func (f *FalcoAlertsSubscriber) ingestEventFromTarianRuleSpawnedProcessAlert(res *outputs.Response, pod *v1.Pod) (*tarianpb.Event, error) {
	outputFields := res.GetOutputFields()
	k8sPodName := outputFields["k8s.pod.name"]
	k8sNsName := outputFields["k8s.ns.name"]

	var t *timestamppb.Timestamp
	if res.GetTime() != nil {
		t = res.GetTime()
	} else {
		t = timestamppb.Now()
	}

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
	pid, err := strconv.Atoi(outputFields["proc.pid"])
	if err != nil {
		logger.Warnw("expected proc.pid to be int value, but got non-int", "proc.pid", outputFields["proc.pid"])
	}
	violatedProcesses[0] = &tarianpb.Process{Pid: int32(pid), Name: outputFields["proc.name"]}

	event := &tarianpb.Event{
		Type:            tarianpb.EventTypeFalcoAlert,
		ClientTimestamp: t,
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
		logger.Errorw("error while reporting falco spawned process", "err", err)
	} else {
		logger.Debugw("ingest event response", "response", response)
	}

	return event, err
}

func (f *FalcoAlertsSubscriber) getConstraintsMatchingLabels(labels map[string]string) []*tarianpb.Constraint {
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

func (f *FalcoAlertsSubscriber) validateProcessAgainstConstraints(processName string, constraints []*tarianpb.Constraint) bool {
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
				logger.Errorw("can not compile regex", "err", err)
				continue
			}

			logger.Debugw("looking for running processes that violate regex", "expr", rgx.String())

			if rgx.MatchString(processName) {
				return true
			}
		}
	}

	return false
}

func (f *FalcoAlertsSubscriber) ingestEventFromGenericFalcoAlert(res *outputs.Response) (*tarianpb.Event, error) {
	var t *timestamppb.Timestamp
	if res.GetTime() != nil {
		t = res.GetTime()
	} else {
		t = timestamppb.Now()
	}

	outputFields := res.GetOutputFields()
	var pod *tarianpb.Pod
	if outputFields != nil {
		pod = &tarianpb.Pod{
			Name:      outputFields["k8s.pod.name"],
			Namespace: outputFields["k8s.ns.name"],
		}
	}

	event := &tarianpb.Event{
		Type:            tarianpb.EventTypeFalcoAlert,
		ClientTimestamp: t,
		Targets: []*tarianpb.Target{
			{
				Pod: pod,
				FalcoAlert: &tarianpb.FalcoAlert{
					Rule:         res.GetRule(),
					Priority:     tarianpb.FalcoPriority(res.GetPriority()),
					Output:       res.GetOutput(),
					OutputFields: res.GetOutputFields(),
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
		logger.Errorw("error while reporting falco alerts", "err", err)
	} else {
		logger.Debugw("ingest event response", "response", response)
	}

	return event, err
}
