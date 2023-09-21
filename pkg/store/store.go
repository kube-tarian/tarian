// Package store provides interfaces for storing objects
package store

// Set is a struct that holds instances of various stores for different types of objects.
type Set struct {
	ActionStore     ActionStore
	ConstraintStore ConstraintStore
	EventStore      EventStore
}
