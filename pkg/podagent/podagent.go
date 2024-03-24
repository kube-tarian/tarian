// Package podagent provides tarian pod agent functionality
package podagent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PodAgent represents the Tarian pod agent.
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
	constraintsInitialized bool

	fileValidationInterval time.Duration

	cancelFunc context.CancelFunc
	cancelCtx  context.Context
	logger     *logrus.Logger

	enableRegisterFiles     bool
	registerFilePaths       []string
	registerFileIgnorePaths []string
}

// NewPodAgent creates a new instance of the PodAgent.
//
// Parameters:
//   - logger: The logger instance.
//   - clusterAgentAddress: The address of the cluster agent.
//
// Returns:
//   - *PodAgent: A new instance of the PodAgent.
func NewPodAgent(logger *logrus.Logger, clusterAgentAddress string) Agent {
	ctx, cancel := context.WithCancel(context.Background())

	return &PodAgent{
		cancelCtx:              ctx,
		cancelFunc:             cancel,
		logger:                 logger,
		constraintsInitialized: false,
		clusterAgentAddress:    clusterAgentAddress,
	}
}

// SetPodLabels sets the labels for the pod.
//
// Parameters:
//   - labels: The labels to set for the pod.
func (p *PodAgent) SetPodLabels(labels []*tarianpb.Label) {
	p.podLabels = labels
	p.logger.Debug("pod label(s) set to: ", p.podLabels)
}

// SetPodName sets the name of the pod.
//
// Parameters:
//   - name: The name of the pod.
func (p *PodAgent) SetPodName(name string) {
	p.podName = name
	p.logger.Debug("pod name set to: ", p.podName)
}

// SetPodUID sets the UID of the pod.
//
// Parameters:
//   - uid: The UID of the pod.
func (p *PodAgent) SetPodUID(uid string) {
	p.podUID = uid
	p.logger.Debug("pod UID set to: ", p.podUID)
}

// SetNamespace sets the namespace of the pod.
//
// Parameters:
//   - namespace: The namespace of the pod.
func (p *PodAgent) SetNamespace(namespace string) {
	p.namespace = namespace
	p.logger.Debug("pod namespace set to: ", p.namespace)
}

// SetFileValidationInterval sets the interval for file validation.
//
// Parameters:
//   - t: The duration for file validation interval.
func (p *PodAgent) SetFileValidationInterval(t time.Duration) {
	p.fileValidationInterval = t
	p.logger.Debug("file validation interval set to: ", p.fileValidationInterval)
}

// EnableRegisterFiles enables file registration.
func (p *PodAgent) EnableRegisterFiles() {
	p.enableRegisterFiles = true
	p.logger.Debug("file registration enabled")
}

// SetRegisterFilePaths sets the file paths to register.
//
// Parameters:
//   - paths: The file paths to register.
func (p *PodAgent) SetRegisterFilePaths(paths []string) {
	p.registerFilePaths = paths
	p.logger.Debug("file paths to register set to: ", p.registerFilePaths)
}

// SetRegisterFileIgnorePaths sets the file paths to ignore during registration.
//
// Parameters:
//   - paths: The file paths to ignore during registration.
func (p *PodAgent) SetRegisterFileIgnorePaths(paths []string) {
	p.registerFileIgnorePaths = paths
	p.logger.Debug("file paths to ignore during registration set to: ", p.registerFileIgnorePaths)
}

// Dial establishes a connection to the cluster agent.
func (p *PodAgent) Dial() {
	p.logger.Debug("establishing a connection to the cluster agent.....")
	var err error
	p.grpcConn, err = grpc.Dial(p.clusterAgentAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		p.logger.WithError(err).Fatal("couldn't connect to the cluster agent")
	}
	p.configClient = tarianpb.NewConfigClient(p.grpcConn)
	p.eventClient = tarianpb.NewEventClient(p.grpcConn)
}

// GracefulStop stops the PodAgent gracefully.
func (p *PodAgent) GracefulStop() {
	p.logger.Debug("gracefully stopping the pod agent")
	p.cancelFunc()

	if p.grpcConn != nil {
		p.grpcConn.Close()
	}
}

// RunThreatScan starts the threat scan loop.
func (p *PodAgent) RunThreatScan() {
	p.logger.Info("starting threat scan....")
	p.Dial()
	defer p.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		_ = p.loopSyncConstraints(p.cancelCtx)
		wg.Done()
	}()

	go func() {
		_ = p.loopValidateFileChecksums(p.cancelCtx)
		wg.Done()
	}()

	wg.Wait()
}

// RunRegister starts the registration loop.
func (p *PodAgent) RunRegister() {
	p.logger.Info("starting file registration....")
	p.Dial()
	defer p.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		_ = p.loopSyncConstraints(p.cancelCtx)
		wg.Done()
	}()

	if p.enableRegisterFiles {
		wg.Add(1)
		go func() {
			_ = p.loopRegisterFileChecksums(p.cancelCtx)
			wg.Done()
		}()
	}

	wg.Wait()
}

// SetConstraints sets the constraints for the pod.
//
// Parameters:
//   - constraints: The constraints to set for the pod.
func (p *PodAgent) SetConstraints(constraints []*tarianpb.Constraint) {
	p.constraintsLock.Lock()
	defer p.constraintsLock.Unlock()

	p.constraints = constraints
	p.logger.Debugf("constraints %v set", p.constraints)
}

// GetConstraints retrieves the constraints for the pod.
//
// Returns:
//   - []*tarianpb.Constraint: The constraints for the pod.
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

// SyncConstraints retrieves and synchronizes constraints from the cluster agent.
func (p *PodAgent) SyncConstraints() {
	p.logger.Trace("syncing constraints from the cluster agent")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := p.configClient.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{Namespace: p.namespace, Labels: p.podLabels})
	if err != nil {
		p.logger.WithError(err).Fatal("couldn't get constraints from the cluster agent")
	}

	p.logger.WithField("constraints", r.GetConstraints()).Trace("received constraints from the cluster agent")
	cancel()

	p.SetConstraints(r.GetConstraints())

	p.constraintsInitialized = true
}

// loopValidateFileChecksums periodically validates file checksums against constraints.
func (p *PodAgent) loopValidateFileChecksums(ctx context.Context) error {
	for {
		select {
		case <-time.After(p.fileValidationInterval):
		case <-ctx.Done():
			return ctx.Err()
		}

		if !p.constraintsInitialized {
			continue
		}

		violatedFiles := p.validateFileChecksums()

		for _, violation := range violatedFiles {
			p.logger.WithFields(logrus.Fields{
				"file":     violation.name,
				"actual":   violation.actualSha256Sum,
				"expected": violation.expectedSha256Sum,
			}).Warn("found a file that violates checksum")
		}

		if len(violatedFiles) > 0 {
			p.ReportViolatedFilesToClusterAgent(violatedFiles)
		}
	}
}

// violatedFile represents a file that violates checksum.
type violatedFile struct {
	name              string
	expectedSha256Sum string
	actualSha256Sum   string
}

// validateFileChecksums validates file checksums against constraints.
//
// Returns:
//   - map[string]*violatedFile: A map of violated files.
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
			p.logger.WithFields(logrus.Fields{
				"file":             allowedFile.GetName(),
				"allowedSha256Sum": allowedFile.GetSha256Sum(),
			}).Debug("validating file sha256 checksum")

			f, err := os.Open(allowedFile.GetName())
			if err != nil {
				p.logger.WithFields(logrus.Fields{
					"file":  allowedFile.GetName(),
					"error": err,
				}).Error("can not open file to check the sha256 checksum")
			}

			s256 := sha256.New()
			if _, err := io.Copy(s256, f); err != nil {
				p.logger.WithFields(logrus.Fields{
					"file":  allowedFile.GetName(),
					"error": err,
				}).Error("can not read file to check the sha256 checksum")
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

// ReportViolatedFilesToClusterAgent reports violated files to the cluster agent.
//
// Parameters:
//   - violatedFiles: A map of violated files.
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
		p.logger.WithError(err).Error("couldn't report violation events to the cluster agent")
	} else {
		p.logger.WithField("response", response).Debug("ingest event response")
	}
}

// matchLabelsFromLabels converts labels to match labels.
//
// Parameters:
//   - labels: The labels to convert.
//
// Returns:
//   - []*tarianpb.MatchLabel: The match labels.
func matchLabelsFromLabels(labels []*tarianpb.Label) []*tarianpb.MatchLabel {
	matchLabels := make([]*tarianpb.MatchLabel, len(labels))

	i := 0
	for _, l := range labels {
		matchLabels[i] = &tarianpb.MatchLabel{Key: l.Key, Value: l.Value}
		i++
	}

	return matchLabels
}

// loopRegisterFileChecksums periodically registers file checksums.
func (p *PodAgent) loopRegisterFileChecksums(ctx context.Context) error {
	for {
		_ = p.registerFileChecksums(ctx)

		select {
		case <-time.After(p.fileValidationInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// registerFileChecksums registers file checksums.
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

			for _, ignoredPattern := range p.registerFileIgnorePaths {
				if matched, _ := filepath.Match(ignoredPattern, path); matched {
					return nil
				}
			}

			fd, err2 := os.Open(path)
			if err2 != nil {
				p.logger.WithError(err2).Warn("can not open file to check the sha256 checksum")
			}

			s256 := sha256.New()
			if _, err := io.Copy(s256, fd); err != nil {
				p.logger.WithFields(logrus.Fields{
					"file":  path,
					"error": err,
				}).Error("can not read file to check the sha256 checksum")
			}

			actualSha256Sum := fmt.Sprintf("%x", s256.Sum(nil))
			if expectedSha256Sum, ok := registeredSha256Sums[path]; ok {
				if actualSha256Sum != expectedSha256Sum {
					p.logger.WithFields(logrus.Fields{
						"name":       path,
						"old_sha256": expectedSha256Sum,
						"new_sha256": actualSha256Sum,
					}).Info("found violated file during auto registration, going to replace with the new checksum")

					pathSha := sha256.New()
					pathSha.Write([]byte(path))
					pathShaStr := fmt.Sprintf("%x", pathSha.Sum(nil))[:10]

					err := p.deleteConstraintByNamePrefix(p.podName + "-" + pathShaStr + "-")
					if err != nil {
						p.logger.WithFields(logrus.Fields{
							"name":  path,
							"error": err,
						}).Error("error while deleting constraint with previous sha256Sum")
					}

					response, err := p.createConstraintWithFileRule(p.podName+"-"+pathShaStr+"-"+actualSha256Sum[:10], path, actualSha256Sum)
					if err != nil {
						p.logger.WithFields(logrus.Fields{
							"name":  path,
							"error": err,
						}).Error("error while registering file constraint")
					} else {
						p.logger.WithField("response", response).Debug("add constraint response")
					}
				}
			} else {
				p.logger.WithFields(logrus.Fields{
					"name":   path,
					"sha256": actualSha256Sum,
				}).Info("found new file, going to register")

				pathSha := sha256.New()
				pathSha.Write([]byte(path))
				pathShaStr := fmt.Sprintf("%x", pathSha.Sum(nil))[:10]

				response, err := p.createConstraintWithFileRule(p.podName+"-"+pathShaStr+"-"+actualSha256Sum[:10], path, actualSha256Sum)

				if err != nil {
					p.logger.WithFields(logrus.Fields{
						"name":  path,
						"error": err,
					}).Error("error while registering file constraint")
				} else {
					p.logger.WithField("response", response).Debug("add constraint response")
				}
			}

			return nil
		})

		if err != nil {
			p.logger.WithFields(logrus.Fields{
				"path":  registerFilePath,
				"error": err,
			}).Error("error while traversing registerFilePaths")
		}
	}

	return nil
}

// createConstraintWithFileRule creates a constraint with a file rule.
//
// Parameters:
//   - constraintName: The name of the constraint.
//   - path: The file path for the rule.
//   - sha256Sum: The SHA256 checksum for the rule.
//
// Returns:
//   - *tarianpb.AddConstraintResponse: The response from adding the constraint.
//   - error: An error if there is an issue adding the constraint.
func (p *PodAgent) createConstraintWithFileRule(constraintName, path, sha256Sum string) (*tarianpb.AddConstraintResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &tarianpb.AddConstraintRequest{
		Constraint: &tarianpb.Constraint{
			Kind:      tarianpb.KindConstraint,
			Namespace: p.namespace,
			Name:      constraintName,
			Selector: &tarianpb.Selector{
				MatchLabels: matchLabelsFromLabels(p.podLabels),
			},
			AllowedFiles: []*tarianpb.AllowedFileRule{{Name: path, Sha256Sum: &sha256Sum}},
		},
	}

	response, err := p.configClient.AddConstraint(ctx, req)

	return response, err
}

// deleteConstraintByNamePrefix deletes constraints by name prefix.
//
// Parameters:
//   - prefix: The prefix of the constraint names to delete.
//
// Returns:
//   - error: An error if there is an issue deleting constraints.
func (p *PodAgent) deleteConstraintByNamePrefix(prefix string) error {
	for _, c := range p.constraints {
		if strings.HasPrefix(c.GetName(), prefix) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_, err := p.configClient.RemoveConstraint(ctx, &tarianpb.RemoveConstraintRequest{Namespace: p.namespace, Name: c.GetName()})
			cancel()

			if err != nil {
				return err
			}
		}
	}

	return nil
}
