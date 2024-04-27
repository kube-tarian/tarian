package nodeagent

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/intelops/tarian-detector/pkg/detector"
	ebpf "github.com/intelops/tarian-detector/pkg/eBPF"
	"github.com/intelops/tarian-detector/pkg/eventparser"
	"github.com/intelops/tarian-detector/tarian"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/scylladb/go-set/strset"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ThreatScanAnnotation is the annotation key used to enable threat scans on pods.
const ThreatScanAnnotation = "pod-agent.k8s.tarian.dev/threat-scan"

// RegisterAnnotation is the annotation key used to register pods with the cluster agent.
const RegisterAnnotation = "pod-agent.k8s.tarian.dev/register"

// NodeAgent represents the node agent responsible for managing constraints and validating processes on a node.
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
	logger     *logrus.Logger

	enableAddConstraint bool
	nodeName            string
}

// NewNodeAgent creates a new instance of the NodeAgent.
//
// Parameters:
//   - logger: The logger instance for logging.
//   - clusterAgentAddress: The address of the cluster agent to connect to.
//
// Returns:
//   - *NodeAgent: A new NodeAgent instance.
func NewNodeAgent(logger *logrus.Logger, clusterAgentAddress string) *NodeAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &NodeAgent{
		clusterAgentAddress:    clusterAgentAddress,
		cancelCtx:              ctx,
		cancelFunc:             cancel,
		constraintsInitialized: false,
		logger:                 logger,
	}
}

// EnableAddConstraint sets whether the NodeAgent should enable adding new constraints.
//
// Parameters:
//   - enabled: A boolean value indicating whether to enable adding constraints.
func (n *NodeAgent) EnableAddConstraint(enabled bool) {
	n.enableAddConstraint = enabled
}

// SetNodeName sets the name of the node.
//
// Parameters:
//   - name: The name of the node.
func (n *NodeAgent) SetNodeName(name string) {
	n.nodeName = name
}

// Dial establishes a gRPC connection to the cluster agent.
func (n *NodeAgent) Dial() {
	var err error
	n.grpcConn, err = grpc.Dial(n.clusterAgentAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	n.configClient = tarianpb.NewConfigClient(n.grpcConn)
	n.eventClient = tarianpb.NewEventClient(n.grpcConn)

	if err != nil {
		n.logger.WithError(err).Fatal("couldn't connect to the cluster agent")
	}
}

// GracefulStop stops the NodeAgent gracefully.
func (n *NodeAgent) GracefulStop() {
	n.cancelFunc()
}

// Run starts the NodeAgent and runs its main loops.
func (n *NodeAgent) Run() {
	n.Dial()
	defer n.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		_ = n.loopSyncConstraints(n.cancelCtx)
		wg.Done()
	}()

	go func() {
		_ = n.loopTarianDetectorReadEvents(n.cancelCtx)
		wg.Done()
	}()

	wg.Wait()
}

// SetConstraints sets the constraints for the NodeAgent.
//
// Parameters:
//   - constraints: A slice of constraints to set.
func (n *NodeAgent) SetConstraints(constraints []*tarianpb.Constraint) {
	n.constraintsLock.Lock()
	defer n.constraintsLock.Unlock()

	n.constraints = constraints
}

// GetConstraints returns the current constraints of the NodeAgent.
//
// Returns:
//   - []*tarianpb.Constraint: A slice of constraints.
func (n *NodeAgent) GetConstraints() []*tarianpb.Constraint {
	return n.constraints
}

// loopSyncConstraints continuously synchronizes constraints with the cluster agent.
//
// Parameters:
//   - ctx: The context for the loop.
//
// Returns:
//   - error: An error, if any, encountered during the loop.
func (n *NodeAgent) loopSyncConstraints(ctx context.Context) error {
	for {
		n.SyncConstraints()

		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return fmt.Errorf("nodeagent: %w", ctx.Err())
		}
	}
}

// SyncConstraints retrieves constraints from the cluster agent and updates the NodeAgent's constraints.
func (n *NodeAgent) SyncConstraints() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := n.configClient.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{})

	if err != nil {
		n.logger.WithError(err).Fatal("couldn't get constraints from the cluster agent")
	}

	n.logger.WithField("constraints", r.GetConstraints()).Trace("received constraints from the cluster agent")
	cancel()

	n.SetConstraints(r.GetConstraints())

	n.constraintsInitialized = true
}

// loopTarianDetectorReadEvents reads events from the Tarian detector and sends them to the cluster agent.
//
// ctx context.Context
// error
func (n *NodeAgent) loopTarianDetectorReadEvents(ctx context.Context) error {
	// Create a PodWatcher to watch for Pods on the node.
	podWatcher, err := n.setupPodWatcher()
	if err != nil {
		return err
	}
	podWatcher.Start()

	eventsDetector, tarianDetector, tarianEbpfModule, err := n.setupEventsDetector()
	if err != nil {
		return err
	}

	// Start eventsDetector and defer Close
	err = eventsDetector.Start()
	if err != nil {
		n.logger.Errorf("error while starting tarian detector: %v", err)
		return fmt.Errorf("error while starting tarian-detector: %w", err)
	}
	defer eventsDetector.Close()

	// Attaches tarian module programs to the kernel
	err = tarianEbpfModule.Attach(tarianDetector)
	if err != nil {
		return fmt.Errorf("error while attaching tarian-detector: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			event, err := eventsDetector.ReadAsInterface()
			if err != nil {
				n.logger.Errorf("tarian-detector: error while read event: %v", err)
				continue
			}

			if event.EventId == "" {
				continue
			}

			pid := event.HostProcessId

			// Retrieve the container ID.
			containerID, err := procsContainerID(pid)
			if err != nil {
				continue
			}

			if containerID == "" {
				continue
			}

			detectionDataType := event.EventId

			// Find the corresponding Kubernetes Pod.
			pod := podWatcher.FindPod(containerID)
			if pod == nil {
				continue
			}

			if detectionDataType == "sys_execve_entry" {
				execEvent, err2 := n.execEventFromTarianDetector(event, containerID, pod)
				if err2 != nil {
					n.logger.WithField("err", err2).Error("tarian-detector: error while converting tarian-detector to execEvent")
				}

				if execEvent != nil {
					err3 := n.handleExecEvent(execEvent)
					if err3 != nil {
						n.logger.WithField("err", err3).Error("node-agent: error while handling exec event")
					}
				}

				n.logger.WithField("execEvent", execEvent).WithField("event", event).Info("handled exec event")
			}

			byteData, err := json.Marshal(event)
			if err != nil {
				n.logger.Error("tarian-detector: error while marshaling event", "err", err)
				continue
			}

			go n.SendDetectionEventToClusterAgent(detectionDataType, string(byteData))
		}
	}
}

func (n *NodeAgent) setupEventsDetector() (*detector.EventsDetector, *ebpf.Handler, *ebpf.Module, error) {
	tarianEbpfModule, err := tarian.GetModule()
	if err != nil {
		n.logger.Errorf("error while get tarian-detector ebpf module: %v", err)
		return nil, nil, nil, fmt.Errorf("error while getting tarian-detector ebpf module: %w", err)
	}

	// Temporarily skip read and write syscalls until tarian-detector
	// is able to reduce the volume of events generated
	for _, p := range tarianEbpfModule.GetPrograms() {
		i, err := p.GetName().Info()
		if err != nil {
			continue
		}

		if strings.Contains(i.Name, "tdf_write") || strings.Contains(i.Name, "tdf_read") {
			p.Disable()
		}
	}

	tarianDetector, err := tarianEbpfModule.Prepare()
	if err != nil {
		n.logger.Errorf("error while prepare tarian-detector: %v", err)
		return nil, nil, nil, fmt.Errorf("error while preparing tarian-detector: %w", err)
	}

	// Instantiate event detectors
	eventsDetector := detector.NewEventsDetector()

	// Add ebpf programs to detectors
	eventsDetector.Add(tarianDetector)

	// Attaches tarian module programs to the kernel
	err = tarianEbpfModule.Attach(tarianDetector)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error while preparing tarian-detector: %w", err)
	}

	return eventsDetector, tarianDetector, tarianEbpfModule, nil
}

func (n *NodeAgent) setupPodWatcher() (*PodWatcher, error) {
	// Get in-cluster configuration for Kubernetes.
	config, err := rest.InClusterConfig()
	if err != nil {
		n.logger.Errorf("error while creating k8s client config: %v", err)
		return nil, fmt.Errorf("error while creating k8s client config: %w", err)
	}

	// Create a Kubernetes client.
	k8sClient := kubernetes.NewForConfigOrDie(config)

	watcher, err := NewPodWatcher(n.logger, k8sClient, n.nodeName)
	if err != nil {
		n.logger.Errorf("error while starting pod-watcher: %v", err)
		return nil, fmt.Errorf("error while starting pod-watcher: %w", err)
	}

	return watcher, nil
}

func (n *NodeAgent) execEventFromTarianDetector(bpfEvt eventparser.TarianDetectorEvent, containerID string, pod *corev1.Pod) (*ExecEvent, error) {
	pid := bpfEvt.HostProcessId

	var podName string
	var podUID string
	var namespace string
	var podLabels map[string]string
	var podAnnotations map[string]string

	podName = pod.GetName()
	podUID = string(pod.GetUID())
	namespace = pod.GetNamespace()
	podLabels = pod.GetLabels()
	podAnnotations = pod.GetAnnotations()

	execFileName := bpfEvt.Executable
	for _, c := range bpfEvt.Context {
		if c.Name == "filename" {
			execFileName = c.Value
			break
		}
	}

	// Running on kubernetes, bpfEvt["processName"] contains `runc:[2:INIT]`
	// So, we take the command from the executable filename instead.
	command := filepath.Base(execFileName)

	// Create an ExecEvent and send it to the events channel.
	execEvent := &ExecEvent{
		Pid:               pid,
		Filename:          execFileName,
		Command:           command,
		ContainerID:       containerID,
		K8sPodName:        podName,
		K8sPodUID:         podUID,
		K8sNamespace:      namespace,
		K8sPodLabels:      podLabels,
		K8sPodAnnotations: podAnnotations,
	}

	return execEvent, nil
}

func (n *NodeAgent) handleExecEvent(evt *ExecEvent) error {
	if !n.constraintsInitialized {
		return nil
	}

	_, threatScanAnnotationPresent := evt.K8sPodAnnotations[ThreatScanAnnotation]
	registerAnnotationValue, registerAnnotationPresent := evt.K8sPodAnnotations[RegisterAnnotation]
	if !threatScanAnnotationPresent && !registerAnnotationPresent {
		return nil
	}

	// Pod has a register annotation but the cluster disables registration
	if registerAnnotationPresent && !n.enableAddConstraint {
		return nil
	}

	violation := n.ValidateProcess(evt)
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
			n.logger.WithField("comm", evt).Debug("violated process detected, going to register")
			go n.RegisterViolationsAsNewConstraint(violation)
		} else {
			n.logger.WithField("comm", evt).Debug("violated process detected")
			go n.ReportViolationsToClusterAgent(violation)
		}
	}

	return nil
}

// SendDetectionEventToClusterAgent sends a detection event to the cluster agent.
//
// It takes two parameters: detectionDataType of type string, and detectionData of type string.
func (n *NodeAgent) SendDetectionEventToClusterAgent(detectionDataType, detectionData string) {
	req := tarianpb.IngestEventRequest{
		Event: &tarianpb.Event{
			Type:            tarianpb.EventTypeDetection,
			ClientTimestamp: timestamppb.New(time.Now()),
			Targets: []*tarianpb.Target{
				{
					DetectionDataType: detectionDataType,
					DetectionData:     detectionData,
				},
			},
		},
	}

	resp, err := n.eventClient.IngestEvent(context.Background(), &req)
	if err != nil {
		n.logger.Error("error while sending detection events ", "err ", err)
	} else {
		n.logger.WithField("response", resp).Trace("ingest event response", "response", resp)
	}
}

// ValidateProcess validates a process event against constraints.
//
// Parameters:
//   - evt: The ExecEvent to validate.
//
// Returns:
//   - *ProcessViolation: A ProcessViolation if the process violates constraints, or nil if it is allowed.
func (n *NodeAgent) ValidateProcess(evt *ExecEvent) *ProcessViolation {
	// Ignore empty pods
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
				n.logger.WithError(err).Error("can not compile regex")
				continue
			}

			n.logger.WithField("expr", rgx.String()).Debug("validating process against regex")

			if rgx.MatchString(evt.Command) {
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

// constraintMatchesPod checks if a constraint matches a pod based on namespace and labels.
//
// Parameters:
//   - constraint: The constraint to check.
//   - evt: The ExecEvent representing the pod event.
//
// Returns:
//   - bool: True if the constraint matches the pod, otherwise false.
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

// ProcessViolation represents a process that violates constraints.
type ProcessViolation struct {
	ExecEvent
}

// ReportViolationsToClusterAgent reports process violations to the cluster agent.
//
// Parameters:
//   - violation: The ProcessViolation to report.
func (n *NodeAgent) ReportViolationsToClusterAgent(violation *ProcessViolation) {
	violatedProcesses := make([]*tarianpb.Process, 1)

	processName := violation.Command
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
		n.logger.WithError(err).Error("error while reporting violation events")
	} else {
		n.logger.WithField("response", response).Debug("ingest event response")
	}
}

// RegisterViolationsAsNewConstraint registers process violations as new constraints with the cluster agent.
//
// Parameters:
//   - violation: The ProcessViolation to register.
func (n *NodeAgent) RegisterViolationsAsNewConstraint(violation *ProcessViolation) {
	k8sPodName := violation.K8sPodName
	k8sNsName := violation.K8sNamespace

	procName := violation.Command
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
		n.logger.WithError(err).WithField("response", response).Error("error while registering process constraint")
	} else {
		n.logger.WithField("response", response).Debug("add constraint response")
	}
}

// matchLabelsFromPodLabels converts a map of labels to a slice of MatchLabel protobufs.
//
// Parameters:
//   - labels: The map of labels to convert.
//
// Returns:
//   - []*tarianpb.MatchLabel: A slice of MatchLabel protobufs.
func matchLabelsFromPodLabels(labels map[string]string) []*tarianpb.MatchLabel {
	matchLabels := make([]*tarianpb.MatchLabel, len(labels))

	i := 0
	for k, v := range labels {
		matchLabels[i] = &tarianpb.MatchLabel{Key: k, Value: v}
		i++
	}

	return matchLabels
}
