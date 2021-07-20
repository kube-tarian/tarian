package store

import "github.com/devopstoday11/tarian/pkg/tarianpb"

type ConstraintStore interface {
	GetAll() ([]*tarianpb.Constraint, error)
	FindByNamespace(namespace string) ([]*tarianpb.Constraint, error)
	Add(*tarianpb.Constraint) error
}
