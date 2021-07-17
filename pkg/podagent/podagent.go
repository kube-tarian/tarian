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

func (p *PodAgent) loopSyncConstraints() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		r, err := p.client.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{Namespace: "default"})

		if err != nil {
			logger.Errorw("error while getting constraints from the cluster agent", "err", err)
		}

		logger.Infow("received constraints from the cluster agent", "constraint", r.GetConstraints())
		cancel()

		p.constraintsLock.Lock()
		p.constraints = r.GetConstraints()
		p.constraintsLock.Unlock()

		time.Sleep(3 * time.Second)
	}
}

func (p *PodAgent) loopWatchProcesses() {
	for {
		processes, _ := process.Processes()

		p.constraintsLock.RLock()

		violations := []*process.Process{}

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
						violations = append(violations, process)
					}
				}
			}
		}

		p.constraintsLock.RUnlock()

		// Currently limit the result to 5
		// TODO: make it configurable
		for _, violation := range violations[:5] {
			name, err := violation.Name()

			if err != nil {
				logger.Errorw("can not read process name", "err", err)
				continue
			}

			logger.Infow("found process that violate regex", "process", name)
		}

		time.Sleep(3 * time.Second)
	}
}
