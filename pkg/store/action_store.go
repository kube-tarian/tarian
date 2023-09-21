package store

import "github.com/kube-tarian/tarian/pkg/tarianpb"

// ActionStore is an interface for storing and retrieving actions.
type ActionStore interface {
	// GetAll retrieves all actions stored in the ActionStore.
	GetAll() ([]*tarianpb.Action, error)

	// FindByNamespace retrieves actions in the specified namespace.
	// Parameters:
	// - namespace: The namespace in which to search for actions.
	FindByNamespace(namespace string) ([]*tarianpb.Action, error)

	// NamespaceAndNameExist checks if an action with the given namespace and name already exists.
	// Parameters:
	// - namespace: The namespace of the action.
	// - name: The name of the action.
	// Returns:
	// - bool: true if an action with the specified namespace and name exists; otherwise, false.
	// - error: An error if the check encounters an issue.
	NamespaceAndNameExist(namespace, name string) (bool, error)

	// Add adds a new action to the ActionStore.
	// Parameters:
	// - action: The action to be added.
	// Returns:
	// - error: An error if adding the action fails.
	Add(action *tarianpb.Action) error

	// RemoveByNamespaceAndName removes an action with the specified namespace and name from the ActionStore.
	// Parameters:
	// - namespace: The namespace of the action to be removed.
	// - name: The name of the action to be removed.
	// Returns:
	// - error: An error if removing the action fails.
	RemoveByNamespaceAndName(namespace, name string) error
}
