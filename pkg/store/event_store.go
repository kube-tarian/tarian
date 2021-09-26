package store

import "github.com/kube-tarian/tarian/pkg/tarianpb"

type EventStore interface {
	GetAll(limit uint) ([]*tarianpb.Event, error)
	FindByNamespace(namespace string, limit uint) ([]*tarianpb.Event, error)
	FindWhereAlertNotSent() ([]*tarianpb.Event, error)
	Add(*tarianpb.Event) error
	UpdateAlertSent(uid string) error
}
