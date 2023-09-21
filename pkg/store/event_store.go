package store

import "github.com/kube-tarian/tarian/pkg/tarianpb"

// EventStore is an interface for storing and retrieving events.
type EventStore interface {
	// GetAll retrieves all events stored in the EventStore with an optional limit on the number of events.
	// Parameters:
	// - limit: The maximum number of events to retrieve.
	GetAll(limit uint) ([]*tarianpb.Event, error)

	// FindByNamespace retrieves events in the specified namespace with an optional limit on the number of events.
	// Parameters:
	// - namespace: The namespace in which to search for events.
	// - limit: The maximum number of events to retrieve.
	FindByNamespace(namespace string, limit uint) ([]*tarianpb.Event, error)

	// FindWhereAlertNotSent retrieves events that have not been sent as alerts.
	// Returns:
	// - []*tarianpb.Event: The list of events that have not been sent as alerts.
	// - error: An error if the retrieval encounters an issue.
	FindWhereAlertNotSent() ([]*tarianpb.Event, error)

	// Add adds a new event to the EventStore.
	// Parameters:
	// - event: The event to be added.
	// Returns:
	// - error: An error if adding the event fails.
	Add(event *tarianpb.Event) error

	// UpdateAlertSent updates the alert sent status of an event with the specified UID.
	// Parameters:
	// - uid: The UID of the event to update.
	// Returns:
	// - error: An error if updating the alert status fails.
	UpdateAlertSent(uid string) error
}
