// Package store provides interfaces for storing objects
package store

type StoreSet struct {
	ActionStore     ActionStore
	ConstraintStore ConstraintStore
	EventStore      EventStore
}
