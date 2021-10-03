package store

import "github.com/kube-tarian/tarian/pkg/tarianpb"

type ActionStore interface {
	GetAll() ([]*tarianpb.Action, error)
	FindByNamespace(namespace string) ([]*tarianpb.Action, error)
	NamespaceAndNameExist(namespace, name string) (bool, error)
	Add(*tarianpb.Action) error
	RemoveByNamespaceAndName(namespace, name string) error
}
