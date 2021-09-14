package podagent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	podName             string
	podUID              string
	podLabels           []*tarianpb.Label
	namespace           string

	constraints            []*tarianpb.Constraint
	constraintsLock        sync.RWMutex
	fileValidationInterval time.Duration

	cancelFunc context.CancelFunc
	cancelCtx  context.Context

	enableRegisterProcesses bool
	enableRegisterFiles     bool
	registerFilePaths       []string
}

func NewPodAgent(clusterAgentAddress string) *PodAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &PodAgent{clusterAgentAddress: clusterAgentAddress, cancelCtx: ctx, cancelFunc: cancel}
}

func (p *PodAgent) SetPodLabels(labels []*tarianpb.Label) {
	p.podLabels = labels
}

func (p *PodAgent) SetPodName(name string) {
	p.podName = name
}

func (p *PodAgent) SetpodUID(uid string) {
	p.podUID = uid
}

func (p *PodAgent) SetNamespace(namespace string) {
	p.namespace = namespace
}

func (p *PodAgent) SetFileValidationInterval(t time.Duration) {
	p.fileValidationInterval = t
}

func (p *PodAgent) EnableRegisterProcesses() {
	p.enableRegisterProcesses = true
}

func (p *PodAgent) EnableRegisterFiles() {
	p.enableRegisterFiles = true
}

func (p *PodAgent) SetRegisterFilePaths(paths []string) {
	p.registerFilePaths = paths
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

func (p *PodAgent) RunThreatScan() {
	p.Dial()
	defer p.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		p.loopSyncConstraints(p.cancelCtx)
		wg.Done()
	}()

	go func() {
		p.loopValidateProcesses(p.cancelCtx)
		wg.Done()
	}()

	go func() {
		p.loopValidateFileChecksums(p.cancelCtx)
		wg.Done()
	}()

	wg.Wait()
}

func (p *PodAgent) RunRegister() {
	p.Dial()
	defer p.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		p.loopSyncConstraints(p.cancelCtx)
		wg.Done()
	}()

	if p.enableRegisterProcesses {
		wg.Add(1)
		go func() {
			p.loopRegisterProcesses(p.cancelCtx)
			wg.Done()
		}()
	}

	if p.enableRegisterFiles {
		wg.Add(1)
		go func() {
			p.loopRegisterFileChecksums(p.cancelCtx)
			wg.Done()
		}()
	}

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

	logger.Debugw("received constraints from the cluster agent", "constraint", r.GetConstraints())
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

			logger.Debugw("found process that violates regex", "process", name)
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

func (p *PodAgent) loopValidateFileChecksums(ctx context.Context) error {
	for {
		violatedFiles := p.validateFileChecksums()

		for _, violation := range violatedFiles {
			logger.Debugw("found a file that violates checksum", "file", violation.name, "actual", violation.actualSha256Sum, "expected", violation.expectedSha256Sum)
		}

		if len(violatedFiles) > 0 {
			p.ReportViolatedFilesToClusterAgent(violatedFiles)
		}

		select {
		case <-time.After(p.fileValidationInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *PodAgent) ReportViolationsToClusterAgent(processes map[int32]*Process) {
	violatedProcesses := make([]*tarianpb.Process, len(processes))

	i := 0
	for _, p := range processes {
		violatedProcesses[i] = &tarianpb.Process{Pid: p.GetPid(), Name: p.GetName()}
		i++
	}

	req := &tarianpb.IngestEventRequest{
		Event: &tarianpb.Event{
			Type:            tarianpb.EventTypeViolation,
			ClientTimestamp: timestamppb.Now(),
			Targets: []*tarianpb.Target{
				{
					Pod: &tarianpb.Pod{
						Uid:       p.podUID,
						Name:      p.podName,
						Namespace: p.namespace,
						Labels:    p.podLabels,
					},
					ViolatedProcesses: violatedProcesses,
				},
			},
		},
	}

	response, err := p.eventClient.IngestEvent(context.Background(), req)

	if err != nil {
		logger.Errorw("error while reporting violation events", "err", err)
	} else {
		logger.Debugw("ingest event response", "response", response)
	}
}

func (p *PodAgent) ValidateProcesses(processes []*Process) map[int32]*Process {
	p.constraintsLock.RLock()

	// map[pid]*process
	violations := make(map[int32]*Process)
	allowedProcesses := make(map[int32]*Process)

	for _, process := range processes {
		violations[process.GetPid()] = process
	}

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

			logger.Debugw("looking for running processes that violate regex", "expr", rgx.String())

			for _, process := range processes {
				name := process.GetName()

				if err != nil {
					logger.Errorw("can not read process name", "err", err)
					continue
				}

				if rgx.MatchString(name) {
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

type violatedFile struct {
	name              string
	expectedSha256Sum string
	actualSha256Sum   string
}

func (p *PodAgent) validateFileChecksums() map[string]*violatedFile {
	p.constraintsLock.RLock()

	// Copy constraints to a local var to not block SyncConstraints() because this function can run quite long
	constraints := make([]*tarianpb.Constraint, len(p.constraints))
	copy(constraints, p.constraints)

	p.constraintsLock.RUnlock()

	violatedFiles := make(map[string]*violatedFile)
	allowedFiles := make(map[string]struct{})

	for _, constraint := range constraints {
		if constraint.GetAllowedFiles() == nil {
			continue
		}

		for _, allowedFile := range constraint.GetAllowedFiles() {
			if allowedFile.GetName() == "" || allowedFile.GetSha256Sum() == "" {
				continue
			}

			logger.Debugw("validating file sha256 checksum", "file", allowedFile.GetName(), "allowedSha256Sum", allowedFile.GetSha256Sum())

			f, err := os.Open(allowedFile.GetName())
			if err != nil {
				logger.Errorw("can not open file to check the sha256 checksum", "file", allowedFile.GetName(), "err", err)
			}

			s256 := sha256.New()
			if _, err := io.Copy(s256, f); err != nil {
				logger.Errorw("can not read file to check the sha256 checksum", "file", allowedFile.GetName(), "err", err)
			}

			actualSha256Sum := fmt.Sprintf("%x", s256.Sum(nil))

			if actualSha256Sum == allowedFile.GetSha256Sum() {
				allowedFiles[allowedFile.GetName()] = struct{}{}
			} else {
				violated := &violatedFile{
					name:              allowedFile.GetName(),
					actualSha256Sum:   actualSha256Sum,
					expectedSha256Sum: allowedFile.GetSha256Sum(),
				}

				violatedFiles[allowedFile.GetName()] = violated
			}

			f.Close()
		}
	}

	for name := range allowedFiles {
		delete(violatedFiles, name)
	}

	return violatedFiles
}

func (p *PodAgent) ReportViolatedFilesToClusterAgent(violatedFiles map[string]*violatedFile) {
	vf := make([]*tarianpb.ViolatedFile, len(violatedFiles))

	i := 0
	for _, f := range violatedFiles {
		vf[i] = &tarianpb.ViolatedFile{Name: f.name, ActualSha256Sum: f.actualSha256Sum, ExpectedSha256Sum: f.expectedSha256Sum}
		i++
	}

	req := &tarianpb.IngestEventRequest{
		Event: &tarianpb.Event{
			Type:            tarianpb.EventTypeViolation,
			ClientTimestamp: timestamppb.Now(),
			Targets: []*tarianpb.Target{
				{
					Pod: &tarianpb.Pod{
						Uid:       p.podUID,
						Name:      p.podName,
						Namespace: p.namespace,
						Labels:    p.podLabels,
					},
					ViolatedFiles: vf,
				},
			},
		},
	}

	response, err := p.eventClient.IngestEvent(context.Background(), req)

	if err != nil {
		logger.Errorw("error while reporting violation events", "err", err)
	} else {
		logger.Debugw("ingest event response", "response", response)
	}
}

func (p *PodAgent) loopRegisterProcesses(ctx context.Context) error {
	for {
		logger.Debugw("loop register processes")
		ps, _ := psutil.Processes()
		processes := NewProcessesFromPsutil(ps)

		violations := p.ValidateProcesses(processes)

		for _, violation := range violations {
			name := violation.GetName()

			logger.Debugw("found process that violates regex", "process", name)
		}

		if len(violations) > 0 {
			p.RegisterViolationsAsNewConstraint(violations)
		}

		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *PodAgent) RegisterViolationsAsNewConstraint(processes map[int32]*Process) {
	allowedProcessRules := []*tarianpb.AllowedProcessRule{}

	deduplicateProcessName := make(map[string]struct{})

	for _, p := range processes {
		name := regexp.QuoteMeta(p.GetName())
		if _, ok := deduplicateProcessName[name]; ok {
			continue
		}

		allowedProcessRules = append(allowedProcessRules, &tarianpb.AllowedProcessRule{Regex: &name})
		deduplicateProcessName[name] = struct{}{}
	}

	req := &tarianpb.AddConstraintRequest{
		Constraint: &tarianpb.Constraint{
			Kind:      tarianpb.KindConstraint,
			Namespace: p.namespace,
			Name:      p.podName + "-" + strconv.FormatInt(time.Now().UnixNano()/time.Hour.Milliseconds(), 10),
			Selector: &tarianpb.Selector{
				MatchLabels: matchLabelsFromLabels(p.podLabels),
			},
			AllowedProcesses: allowedProcessRules,
		},
	}

	response, err := p.configClient.AddConstraint(context.Background(), req)

	if err != nil {
		logger.Errorw("error while registering process constraint", "err", err)
	} else {
		logger.Debugw("add constraint response", "response", response)
	}
}

func matchLabelsFromLabels(labels []*tarianpb.Label) []*tarianpb.MatchLabel {
	matchLabels := make([]*tarianpb.MatchLabel, len(labels))

	i := 0
	for _, l := range labels {
		matchLabels[i] = &tarianpb.MatchLabel{Key: l.Key, Value: l.Value}
		i++
	}

	return matchLabels
}

func (p *PodAgent) loopRegisterFileChecksums(ctx context.Context) error {
	for {
		p.registerFileChecksums(ctx)

		select {
		case <-time.After(p.fileValidationInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *PodAgent) registerFileChecksums(ctx context.Context) error {
	registeredSha256Sums := make(map[string]string)

	p.constraintsLock.RLock()
	for _, constraint := range p.constraints {
		if constraint.GetAllowedFiles() == nil {
			continue
		}

		for _, allowedFile := range constraint.GetAllowedFiles() {
			if allowedFile.GetName() == "" || allowedFile.GetSha256Sum() == "" {
				continue
			}

			registeredSha256Sums[allowedFile.GetName()] = allowedFile.GetSha256Sum()
		}
	}
	p.constraintsLock.RUnlock()

	for _, registerFilePath := range p.registerFilePaths {
		err := filepath.WalkDir(registerFilePath, func(path string, f fs.DirEntry, err error) error {
			if f.IsDir() {
				return nil
			}

			fd, err2 := os.Open(path)
			if err2 != nil {
				logger.Warnw("can not open file to check the sha256 checksum", "file", path, "err", err)
			}

			s256 := sha256.New()
			if _, err := io.Copy(s256, fd); err != nil {
				logger.Errorw("can not read file to check the sha256 checksum", "file", path, "err", err)
			}

			actualSha256Sum := fmt.Sprintf("%x", s256.Sum(nil))
			if expectedSha256Sum, ok := registeredSha256Sums[path]; ok {
				if actualSha256Sum != expectedSha256Sum {
					// TODO: registered file but checksum doesn't match
					logger.Infow("violated file", "name", path)
				}
			} else {
				logger.Infow("found new file, going to register", "name", path, "checksum", actualSha256Sum)

				pathSha := sha256.New()
				pathSha.Write([]byte(path))
				pathShaStr := fmt.Sprintf("%x", pathSha.Sum(nil))[:10]

				req := &tarianpb.AddConstraintRequest{
					Constraint: &tarianpb.Constraint{
						Kind:      tarianpb.KindConstraint,
						Namespace: p.namespace,
						Name:      p.podName + "-" + pathShaStr + "-" + actualSha256Sum[:10],
						Selector: &tarianpb.Selector{
							MatchLabels: matchLabelsFromLabels(p.podLabels),
						},
						AllowedFiles: []*tarianpb.AllowedFileRule{{Name: path, Sha256Sum: &actualSha256Sum}},
					},
				}

				response, err := p.configClient.AddConstraint(context.Background(), req)

				if err != nil {
					logger.Errorw("error while registering file constraint", "name", path, "err", err)
				} else {
					logger.Debugw("add constraint response", "response", response)
				}
			}

			return nil
		})

		if err != nil {
			logger.Errorw("error while traversing registerFilePaths", "path", registerFilePath, "err", err)
		}
	}

	return nil
}
