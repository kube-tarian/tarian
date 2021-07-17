package store

import "github.com/devopstoday11/tarian/pkg/tarianpb"

type ConstraintStore interface {
	GetAll() ([]*tarianpb.Constraint, error)
	FindByNamespace(namespace string) ([]*tarianpb.Constraint, error)
	Add(*tarianpb.Constraint) error
}

type MemoryConstraintStore struct {
	data map[string][]*tarianpb.Constraint
}

func NewMemoryConstraintStore() *MemoryConstraintStore {
	m := &MemoryConstraintStore{data: make(map[string][]*tarianpb.Constraint)}

	regexes := []string{"ssh", "worker", "swap", "scsi", "loop", "gvfs", "idle", "injection", "nvme", "jbd", "snap", "cpu", "soft", "bash", "integrity", "kcryptd", "krfcommd", "kcompactd0", "wpa_supplican", "oom_reaper", "registryd", "migration", "kblockd", "gsd-", "kdevtmpfs", "pipewire"}

	for _, r := range regexes {
		exampleConstraint := tarianpb.Constraint{Namespace: "default", Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}
		allowedProcessRegex := "(.*)" + r + "(.*)"
		exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}
		m.Add(&exampleConstraint)
	}

	return m
}

func (m *MemoryConstraintStore) GetAll() ([]*tarianpb.Constraint, error) {
	allConstraints := []*tarianpb.Constraint{}

	for _, nsConstraints := range m.data {
		allConstraints = append(allConstraints, nsConstraints...)
	}

	return allConstraints, nil
}

func (m *MemoryConstraintStore) FindByNamespace(namespace string) ([]*tarianpb.Constraint, error) {
	allConstraints := []*tarianpb.Constraint{}
	allConstraints = append(allConstraints, m.data[namespace]...)

	return allConstraints, nil
}

func (m *MemoryConstraintStore) Add(constraint *tarianpb.Constraint) error {
	m.data[constraint.GetNamespace()] = append(m.data[constraint.GetNamespace()], constraint)

	return nil
}
