package podagent

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	psutil "github.com/shirou/gopsutil/process"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var logger *zap.SugaredLogger

func init() {
	l, err := zap.NewProduction()

	if err != nil {
		panic("Can not create logger")
	}

	logger = l.Sugar()
}

func SetLogger(l *zap.SugaredLogger) {
	logger = l
}

type PodAgent struct {
	clusterAgentAddress string
	grpcConn            *grpc.ClientConn
	configClient        tarianpb.ConfigClient
	eventClient         tarianpb.EventClient
	podLabels           []*tarianpb.Label
	namespace           string

	constraints     []*tarianpb.Constraint
	constraintsLock sync.RWMutex

	cancelFunc context.CancelFunc
	cancelCtx  context.Context
}

func NewPodAgent(clusterAgentAddress string) *PodAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &PodAgent{clusterAgentAddress: clusterAgentAddress, cancelCtx: ctx, cancelFunc: cancel}
}

func (p *PodAgent) SetPodLabels(labels []*tarianpb.Label) {
	p.podLabels = labels
}

func (p *PodAgent) SetNamespace(namespace string) {
	p.namespace = namespace
}

func (p *PodAgent) Dial() {
	var err error
	p.grpcConn, err = grpc.Dial(p.clusterAgentAddress, grpc.WithInsecure())
	p.configClient = tarianpb.NewConfigClient(p.grpcConn)
	p.eventClient = tarianpb.NewEventClient(p.grpcConn)

	if err != nil {
		logger.Fatalw("couldn't connect to the cluster agent", "err", err)
	}
}

func (p *PodAgent) GracefulStop() {
	p.cancelFunc()

	if p.grpcConn != nil {
		p.grpcConn.Close()
	}
}

func (p *PodAgent) Run() {
	p.Dial()
	defer p.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		p.loopSyncConstraints(p.cancelCtx)
		wg.Done()
	}()

	go func() {
		p.loopValidateProcesses(p.cancelCtx)
		wg.Done()
	}()

	wg.Wait()
}

func (p *PodAgent) SetConstraints(constraints []*tarianpb.Constraint) {
	p.constraintsLock.Lock()
	defer p.constraintsLock.Unlock()

	p.constraints = constraints
}

func (p *PodAgent) GetConstraints() []*tarianpb.Constraint {
	return p.constraints
}

func (p *PodAgent) loopSyncConstraints(ctx context.Context) error {
	for {
		p.SyncConstraints()

		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *PodAgent) SyncConstraints() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := p.configClient.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{Namespace: p.namespace, Labels: p.podLabels})

	if err != nil {
		logger.Errorw("error while getting constraints from the cluster agent", "err", err)
	}

	logger.Infow("received constraints from the cluster agent", "constraint", r.GetConstraints())
	cancel()

	p.SetConstraints(r.GetConstraints())
}

func (p *PodAgent) loopValidateProcesses(ctx context.Context) error {
	for {
		ps, _ := psutil.Processes()
		processes := NewProcessesFromPsutil(ps)

		violations := p.ValidateProcesses(processes)

		for _, violation := range violations {
			name := violation.GetName()

			logger.Infow("found process that violates regex", "process", name)
		}

		if len(violations) > 0 {
			p.ReportViolationsToClusterAgent(violations)
		}

		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *PodAgent) ReportViolationsToClusterAgent(processes map[int32]*Process) {
	violatingProcesses := make([]*tarianpb.Process, len(processes))

	i := 0
	for _, p := range processes {
		violatingProcesses[i] = &tarianpb.Process{Pid: p.GetPid(), Name: p.GetName()}
		i++
	}

	req := &tarianpb.IngestEventRequest{
		Event: &tarianpb.Event{
			Type:            tarianpb.EventTypeViolation,
			ClientTimestamp: timestamppb.Now(),
			Targets: []*tarianpb.Target{
				{
					Pod: &tarianpb.Pod{
						Uid:       "abc-def-ghe",
						Name:      "pod-name-placeholder",
						Namespace: "tarian-system",
						Labels: []*tarianpb.Label{
							{
								Key:   "app",
								Value: "nginx",
							},
						},
					},
					ViolatingProcesses: violatingProcesses,
				},
			},
		},
	}

	response, err := p.eventClient.IngestEvent(context.Background(), req)

	if err != nil {
		logger.Errorw("error while reporting violation events", "err", err)
	} else {
		logger.Infow("ingest event response", "response", response)
	}
}

func (p *PodAgent) ValidateProcesses(processes []*Process) map[int32]*Process {
	p.constraintsLock.RLock()

	// map[pid]*process
	violations := make(map[int32]*Process)
	allowedProcesses := make(map[int32]*Process)

	for _, constraint := range p.constraints {
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

			logger.Infow("looking for running processes that violate regex", "expr", rgx.String())

			for _, process := range processes {
				name := process.GetName()

				if err != nil {
					logger.Errorw("can not read process name", "err", err)
					continue
				}

				if !rgx.MatchString(name) {
					violations[process.GetPid()] = process
				} else {
					allowedProcesses[process.GetPid()] = process
				}
			}
		}
	}

	p.constraintsLock.RUnlock()

	for pid := range allowedProcesses {
		delete(violations, pid)
	}

	return violations
}
