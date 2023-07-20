package nodeagent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/intelops/tarian-detector/pkg/detector"
	"github.com/intelops/tarian-detector/pkg/ebpf/c/process_entry"
	"github.com/intelops/tarian-detector/pkg/ebpf/c/process_exit"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/scylladb/go-set/strset"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ThreatScanAnnotation = "pod-agent.k8s.tarian.dev/threat-scan"
	RegisterAnnotation   = "pod-agent.k8s.tarian.dev/register"
)

type NodeAgent struct {
	clusterAgentAddress string
	grpcConn            *grpc.ClientConn
	configClient        tarianpb.ConfigClient
	eventClient         tarianpb.EventClient

	constraints            []*tarianpb.Constraint
	constraintsLock        sync.RWMutex
	constraintsInitialized bool

	cancelFunc context.CancelFunc
	cancelCtx  context.Context

	enableAddConstraint bool
	nodeName            string
}

func NewNodeAgent(clusterAgentAddress string) *NodeAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &NodeAgent{clusterAgentAddress: clusterAgentAddress, cancelCtx: ctx, cancelFunc: cancel, constraintsInitialized: false}
}

func (n *NodeAgent) EnableAddConstraint(enabled bool) {
	n.enableAddConstraint = enabled
}

func (n *NodeAgent) SetNodeName(name string) {
	n.nodeName = name
}

func (n *NodeAgent) Dial() {
	var err error
	n.grpcConn, err = grpc.Dial(n.clusterAgentAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	n.configClient = tarianpb.NewConfigClient(n.grpcConn)
	n.eventClient = tarianpb.NewEventClient(n.grpcConn)

	if err != nil {
		logger.Fatalw("couldn't connect to the cluster agent", "err", err)
	}
}

func (n *NodeAgent) GracefulStop() {
	n.cancelFunc()
}

func (n *NodeAgent) Run() {
	n.Dial()
	defer n.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		n.loopSyncConstraints(n.cancelCtx)
		wg.Done()
	}()

	go func() {
		n.loopValidateProcesses(n.cancelCtx)
		wg.Done()
	}()

	go func() {
		n.loopTarianDetectorReadEvents(n.cancelCtx)
		wg.Done()
	}()

	wg.Wait()
}

func (n *NodeAgent) SetConstraints(constraints []*tarianpb.Constraint) {
	n.constraintsLock.Lock()
	defer n.constraintsLock.Unlock()

	n.constraints = constraints
}

func (n *NodeAgent) GetConstraints() []*tarianpb.Constraint {
	return n.constraints
}

func (n *NodeAgent) loopSyncConstraints(ctx context.Context) error {
	for {
		n.SyncConstraints()

		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *NodeAgent) SyncConstraints() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := n.configClient.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{})

	if err != nil {
		logger.Errorw("error while getting constraints from the cluster agent", "err", err)
	}

	logger.Debugw("received constraints from the cluster agent", "constraint", r.GetConstraints())
	cancel()

	n.SetConstraints(r.GetConstraints())

	n.constraintsInitialized = true
}

func (n *NodeAgent) loopValidateProcesses(ctx context.Context) error {
	captureExec, err := NewCaptureExec()
	if err != nil {
		logger.Fatal(err)
	}

	captureExec.SetNodeName(n.nodeName)

	execEvent := captureExec.GetEventsChannel()
	go captureExec.Start()

	for {
		select {
		case <-ctx.Done():
			captureExec.Close()
			return ctx.Err()
		case evt := <-execEvent:
			if !n.constraintsInitialized {
				continue
			}

			_, threatScanAnnotationPresent := evt.K8sPodAnnotations[ThreatScanAnnotation]
			registerAnnotationValue, registerAnnotationPresent := evt.K8sPodAnnotations[RegisterAnnotation]
			if !threatScanAnnotationPresent && !registerAnnotationPresent {
				continue
			}

			// Pod has register annotation but the cluster disable registration
			if registerAnnotationPresent && !n.enableAddConstraint {
				continue
			}

			violation := n.ValidateProcess(&evt)
			if violation != nil {
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

				if registerProcess {
					logger.Infow("violated process detected, going to register", "comm", evt.Comm)

					n.RegisterViolationsAsNewConstraint(violation)
				} else {
					logger.Infow("violated process detected", "comm", evt.Comm)

					n.ReportViolationsToClusterAgent(violation)
				}
			}
		}
	}
}

func (n *NodeAgent) ValidateProcess(evt *ExecEvent) *ProcessViolation {
	// Ignore empty pod
	// It usually means a host process
	if evt.K8sNamespace == "" || evt.K8sPodName == "" {
		return nil
	}

	n.constraintsLock.RLock()

	violated := true

out:
	for _, constraint := range n.constraints {
		if constraint.GetAllowedProcesses() == nil {
			continue
		}

		if !constraintMatchesPod(constraint, evt) {
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

			logger.Debugw("validating process againts regex", "expr", rgx.String())

			if rgx.MatchString(evt.Comm) {
				violated = false
				break out
			}
		}
	}

	n.constraintsLock.RUnlock()

	if violated {
		return &ProcessViolation{*evt}
	}

	return nil
}

func constraintMatchesPod(constraint *tarianpb.Constraint, evt *ExecEvent) bool {
	if constraint.GetNamespace() != evt.K8sNamespace {
		return false
	}

	if constraint.GetSelector() == nil || constraint.GetSelector().GetMatchLabels() == nil {
		return true
	}

	constraintLabels := strset.New()
	for _, l := range constraint.GetSelector().GetMatchLabels() {
		constraintLabels.Add(l.GetKey() + "=" + l.GetValue())
	}

	podLabels := strset.New()
	for k, v := range evt.K8sPodLabels {
		podLabels.Add(k + "=" + v)
	}

	return podLabels.IsSubset(constraintLabels)
}

type ProcessViolation struct {
	ExecEvent
}

func (n *NodeAgent) ReportViolationsToClusterAgent(violation *ProcessViolation) {
	violatedProcesses := make([]*tarianpb.Process, 1)

	processName := violation.Comm
	violatedProcesses[0] = &tarianpb.Process{Pid: int32(violation.Pid), Name: processName}

	pbPodLabels := make([]*tarianpb.Label, len(violation.K8sPodLabels))
	for k, v := range violation.K8sPodLabels {
		pbPodLabels = append(pbPodLabels, &tarianpb.Label{Key: k, Value: v})
	}

	req := &tarianpb.IngestEventRequest{
		Event: &tarianpb.Event{
			Type:            tarianpb.EventTypeViolation,
			ClientTimestamp: timestamppb.Now(),
			Targets: []*tarianpb.Target{
				{
					Pod: &tarianpb.Pod{
						Uid:       violation.K8sPodUID,
						Name:      violation.K8sPodName,
						Namespace: violation.K8sNamespace,
						Labels:    pbPodLabels,
					},
					ViolatedProcesses: violatedProcesses,
				},
			},
		},
	}

	response, err := n.eventClient.IngestEvent(context.Background(), req)

	if err != nil {
		logger.Errorw("error while reporting violation events", "err", err)
	} else {
		logger.Debugw("ingest event response", "response", response)
	}
}

func (n *NodeAgent) RegisterViolationsAsNewConstraint(violation *ProcessViolation) {
	k8sPodName := violation.K8sPodName
	k8sNsName := violation.K8sNamespace

	procName := violation.Comm
	allowedProcessRules := []*tarianpb.AllowedProcessRule{{Regex: &procName}}

	podLabels := violation.K8sPodLabels
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

	response, err := n.configClient.AddConstraint(context.Background(), req)

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

func (n *NodeAgent) loopTarianDetectorReadEvents(ctx context.Context) error {
	processEntryDetector := process_entry.NewProcessEntryDetector()
	processExitDetector := process_exit.NewProcessExitDetector()

	eventsDetector := detector.NewEventsDetector()
	eventsDetector.Add(processEntryDetector)
	eventsDetector.Add(processExitDetector)

	err := eventsDetector.Start()
	if err != nil {
		return err
	}

	defer eventsDetector.Close()

	go func() {
		for {
			e, err := eventsDetector.ReadAsInterface()
			if errors.Is(err, os.ErrClosed) {
				break
			}

			if err != nil {
				logger.Errorw("tarian-detector: error on reading as itnerface", "err", err)
				continue
			}

			switch event := e.(type) {
			case *process_entry.EntryEventData:
				detectionDataType := "process_entry.EntryEventData"
				data, err := json.Marshal(event)
				if err != nil {
					logger.Errorw("tarian-detector: error while marshaling json", "err", err, "detectionDataType", detectionDataType)
					continue
				}

				n.SendDetectionEventToClusterAgent(detectionDataType, string(data))
				logger.Infow("tarian-detector: process_entry.EntryEventData", "binary_file_path", event.BinaryFilepath, "pid", event.Pid, "comm", event.Comm)

			case *process_exit.ExitEventData:
				detectionDataType := "process_exit.ExitEventData"
				data, err := json.Marshal(event)
				if err != nil {
					logger.Errorw("tarian-detector: error while marshaling json", "err", err, "detectionDataType", detectionDataType)
					continue
				}

				n.SendDetectionEventToClusterAgent(detectionDataType, string(data))
				logger.Infow("tarian-detector: process_exit.ExitEventData", "pid", event.Pid, "comm", event.Comm)
			}
		}
	}()

	<-ctx.Done()
	return ctx.Err()
}

func (n *NodeAgent) SendDetectionEventToClusterAgent(detectionDataType string, detectionData string) {
	req := &tarianpb.IngestEventRequest{
		Event: &tarianpb.Event{
			Type:            tarianpb.EventTypeDetection,
			ClientTimestamp: timestamppb.Now(),
			Targets: []*tarianpb.Target{
				{
					DetectionDataType: detectionDataType,
					DetectionData:     detectionData,
				},
			},
		},
	}

	response, err := n.eventClient.IngestEvent(context.Background(), req)
	if err != nil {
		logger.Errorw("error while sending detection events", "err", err)
	} else {
		logger.Debugw("ingest event response", "response", response)
	}
}
