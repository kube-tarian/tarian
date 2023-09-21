package store

import "github.com/kube-tarian/tarian/pkg/tarianpb"

// ConstraintStore is an interface for storing and retrieving constraints.
type ConstraintStore interface {
	// GetAll retrieves all constraints stored in the ConstraintStore.
	GetAll() ([]*tarianpb.Constraint, error)

	// FindByNamespace retrieves constraints in the specified namespace.
	// Parameters:
	// - namespace: The namespace in which to search for constraints.
	FindByNamespace(namespace string) ([]*tarianpb.Constraint, error)

	// NamespaceAndNameExist checks if a constraint with the given namespace and name already exists.
	// Parameters:
	// - namespace: The namespace of the constraint.
	// - name: The name of the constraint.
	// Returns:
	// - bool: true if a constraint with the specified namespace and name exists; otherwise, false.
	// - error: An error if the check encounters an issue.
	NamespaceAndNameExist(namespace, name string) (bool, error)

	// Add adds a new constraint to the ConstraintStore.
	// Parameters:
	// - constraint: The constraint to be added.
	// Returns:
	// - error: An error if adding the constraint fails.
	Add(constraint *tarianpb.Constraint) error

	// RemoveByNamespaceAndName removes a constraint with the specified namespace and name from the ConstraintStore.
	// Parameters:
	// - namespace: The namespace of the constraint to be removed.
	// - name: The name of the constraint to be removed.
	// Returns:
	// - error: An error if removing the constraint fails.
	RemoveByNamespaceAndName(namespace, name string) error
}
