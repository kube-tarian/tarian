package podagent

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/shirou/gopsutil/process"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	client              tarianpb.ConfigClient

	constraints     []*tarianpb.Constraint
	constraintsLock sync.RWMutex
}

func NewPodAgent(clusterAgentAddress string) *PodAgent {
	return &PodAgent{clusterAgentAddress: clusterAgentAddress}
}

func (p *PodAgent) Run() {
	var err error
	p.grpcConn, err = grpc.Dial(p.clusterAgentAddress, grpc.WithInsecure(), grpc.WithBlock())
	p.client = tarianpb.NewConfigClient(p.grpcConn)

	if err != nil {
		logger.Fatalw("couldn't connect to the cluster agent", "err", err)
	}

	defer p.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	// TODO: implement graceful shutdown
	go p.loopSyncConstraints()
	go p.loopWatchProcesses()

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

func (p *PodAgent) loopSyncConstraints() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		r, err := p.client.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{Namespace: "default"})

		if err != nil {
			logger.Errorw("error while getting constraints from the cluster agent", "err", err)
		}

		logger.Infow("received constraints from the cluster agent", "constraint", r.GetConstraints())
		cancel()

		p.SetConstraints(r.GetConstraints())

		time.Sleep(3 * time.Second)
	}
}

func (p *PodAgent) loopWatchProcesses() {
	for {
		processes, _ := process.Processes()
		p.ValidateProcesses(processes)

		time.Sleep(3 * time.Second)
	}
}

func (p *PodAgent) ValidateProcesses(processes []*process.Process) {
	p.constraintsLock.RLock()

	// map[pid]*process
	violations := make(map[int32]*process.Process)
	allowedProcesses := make(map[int32]*process.Process)

	for _, constraint := range p.constraints {
		if constraint.GetAllowedProcesses() == nil {
			continue
		}

		for _, allowedProcess := range constraint.GetAllowedProcesses() {
			// matched, err := regexp.MatchString(`foo.*`, "seafood")

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
				name, err := process.Name()

				if err != nil {
					logger.Errorw("can not read process name", "err", err)
					continue
				}

				if !rgx.MatchString(name) {
					violations[process.Pid] = process
				} else {
					allowedProcesses[process.Pid] = process
				}
			}
		}
	}

	p.constraintsLock.RUnlock()

	for pid := range allowedProcesses {
		delete(violations, pid)
	}

	// Currently limit the result to 5
	count := 0

	// TODO: make it configurable
	for _, violation := range violations {
		name, err := violation.Name()

		if err != nil {
			logger.Errorw("can not read process name", "err", err)
			continue
		}

		logger.Infow("found process that violate regex", "process", name)

		count++
		if count > 5 {
			break
		}
	}
}
