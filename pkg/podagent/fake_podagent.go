package podagent

import (
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
)

type fakePodAgent struct {
	logger *logrus.Logger
}

// NewFakePodAgent returns a fake pod agent.
func NewFakePodAgent(logger *logrus.Logger) Agent {
	return &fakePodAgent{
		logger: logger,
	}
}

// Dial implements Agent.
func (f *fakePodAgent) Dial() {
	f.logger.Info("Dial")
}

// EnableRegisterFiles implements Agent.
func (f *fakePodAgent) EnableRegisterFiles() {
	f.logger.Info("EnableRegisterFiles")
}

// GetConstraints implements Agent.
func (f *fakePodAgent) GetConstraints() []*tarianpb.Constraint {
	var regex = "regex-1"
	var hash = "hash-1"
	values := []*tarianpb.Constraint{
		{

			Kind:      "FakeKind",
			Namespace: "test-ns1",
			Name:      "constraint-1",
			Selector: &tarianpb.Selector{
				MatchLabels: []*tarianpb.MatchLabel{
					{Key: "key1", Value: "value1"},
				},
			},
			AllowedProcesses: []*tarianpb.AllowedProcessRule{
				{Regex: &regex},
			},
			AllowedFiles: []*tarianpb.AllowedFileRule{
				{Name: "file-1", Sha256Sum: &hash},
			},
		},
	}
	f.logger.Info(values)
	return values
}

// GracefulStop implements Agent.
func (f *fakePodAgent) GracefulStop() {
	f.logger.Info("GracefulStop")
}

// ReportViolatedFilesToClusterAgent implements Agent.
func (f *fakePodAgent) ReportViolatedFilesToClusterAgent(violatedFiles map[string]*violatedFile) {
	f.logger.Info("ReportViolatedFilesToClusterAgent")
}

// RunRegister implements Agent.
func (f *fakePodAgent) RunRegister() {
	f.logger.Info("RunRegister")
}

// RunThreatScan implements Agent.
func (f *fakePodAgent) RunThreatScan() {
	f.logger.Info("RunThreatScan")
}

// SetConstraints implements Agent.
func (f *fakePodAgent) SetConstraints(constraints []*tarianpb.Constraint) {
	f.logger.Info("SetConstraints")
}

// SetFileValidationInterval implements Agent.
func (f *fakePodAgent) SetFileValidationInterval(t time.Duration) {
	f.logger.Info("SetFileValidationInterval")
}

// SetNamespace implements Agent.
func (f *fakePodAgent) SetNamespace(namespace string) {
	f.logger.Info("SetNamespace")
}

// SetPodLabels implements Agent.
func (f *fakePodAgent) SetPodLabels(labels []*tarianpb.Label) {
	f.logger.Info("SetPodLabels")
}

// SetPodName implements Agent.
func (f *fakePodAgent) SetPodName(name string) {
	f.logger.Info("SetPodName")
}

// SetPodUID implements Agent.
func (f *fakePodAgent) SetPodUID(uid string) {
	f.logger.Info("SetPodUID")
}

// SetRegisterFileIgnorePaths implements Agent.
func (f *fakePodAgent) SetRegisterFileIgnorePaths(paths []string) {
	f.logger.Info("SetRegisterFileIgnorePaths")
}

// SetRegisterFilePaths implements Agent.
func (f *fakePodAgent) SetRegisterFilePaths(paths []string) {
	f.logger.Info("SetRegisterFilePaths")
}

// SyncConstraints implements Agent.
func (f *fakePodAgent) SyncConstraints() {
	f.logger.Info("SyncConstraints")
}
