package dgraphstore

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"golang.org/x/net/context"
)

// DgraphConstraintStore is a store for managing Constraints using Dgraph as the backend.
type DgraphConstraintStore struct {
	dgraphClient *dgo.Dgraph
}

// NewDgraphConstraintStore creates a new DgraphConstraintStore with the provided Dgraph client.
//
// Parameters:
// - dgraphClient: A Dgraph client for database interaction.
//
// Returns:
// - A new instance of DgraphConstraintStore.
func NewDgraphConstraintStore(dgraphClient *dgo.Dgraph) *DgraphConstraintStore {
	d := &DgraphConstraintStore{dgraphClient: dgraphClient}
	return d
}

// GetAll retrieves all constraints from the Dgraph store and returns them as protobuf Constraints.
//
// Returns:
// - An array of protobuf Constraint messages representing all constraints in the store.
// - An error if there was an issue with the database query.
func (d *DgraphConstraintStore) GetAll() ([]*tarianpb.Constraint, error) {
	// Dgraph query to retrieve all constraints.
	q := fmt.Sprintf(`
		{
			constraints(func: type(Constraint)) {
				%s
			}
		}
	`, constraintFields)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tx := d.dgraphClient.NewReadOnlyTxn()
	resp, err := tx.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	var result dgraphConstraintList
	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, err
	}

	constraints := result.toPbConstraints()

	return constraints, nil
}

// Constants for the fields needed in Dgraph query.
const constraintFields = `
	uid
	dgraph.type
	constraint_namespace
	constraint_name
	constraint_selector
	constraint_allowed_processes
	constraint_allowed_files
`

// dgraphConstraintList is a helper struct to unmarshal Dgraph query results.
type dgraphConstraintList struct {
	Constraints []Constraint
}

// toPbConstraints converts Dgraph Constraint entities to protobuf Constraint messages.
//
// Returns:
// - An array of protobuf Constraint messages.
func (da *dgraphConstraintList) toPbConstraints() []*tarianpb.Constraint {
	constraints := []*tarianpb.Constraint{}
	for _, c := range da.Constraints {
		constraint := tarianpb.NewConstraint()
		constraint.Namespace = c.Namespace
		constraint.Name = c.Name
		_ = json.Unmarshal([]byte(c.Selector), &constraint.Selector)
		_ = json.Unmarshal([]byte(c.AllowedProcesses), &constraint.AllowedProcesses)
		_ = json.Unmarshal([]byte(c.AllowedFiles), &constraint.AllowedFiles)

		constraints = append(constraints, constraint)
	}

	return constraints
}

// FindByNamespace retrieves constraints from the Dgraph store by namespace and returns them as protobuf Constraints.
//
// Parameters:
// - namespace: The namespace to filter constraints by.
//
// Returns:
// - An array of protobuf Constraint messages representing matching constraints.
// - An error if there was an issue with the database query.
func (d *DgraphConstraintStore) FindByNamespace(namespace string) ([]*tarianpb.Constraint, error) {
	// Dgraph query to retrieve constraints by namespace.
	q := fmt.Sprintf(`
		query constraintQuery($namespace: string) {
			constraints(func: type(Constraint)) @filter(eq(constraint_namespace, $namespace)) {
				%s
			}
		}
	`, constraintFields)

	tx := d.dgraphClient.NewReadOnlyTxn()
	vars := map[string]string{"$namespace": namespace}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := tx.QueryWithVars(ctx, q, vars)
	if err != nil {
		return nil, err
	}

	var result dgraphConstraintList
	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, err
	}

	constraints := result.toPbConstraints()

	return constraints, nil
}

// NamespaceAndNameExist checks if a constraint with the given namespace and name exists in the Dgraph store.
//
// Parameters:
// - namespace: The namespace of the constraint to check.
// - name: The name of the constraint to check.
//
// Returns:
// - A boolean indicating whether the constraint exists.
// - An error if there was an issue with the database query.
func (d *DgraphConstraintStore) NamespaceAndNameExist(namespace, name string) (bool, error) {
	uid, err := d.findConstraintUIDByNamespaceAndName(namespace, name)

	if err != nil {
		return false, err
	}

	exist := uid != ""
	return exist, err
}

// Add adds a new constraint to the Dgraph store.
//
// Parameters:
// - constraint: The protobuf Constraint message to add to the store.
//
// Returns:
// - An error if there was an issue storing the constraint in the database.
func (d *DgraphConstraintStore) Add(constraint *tarianpb.Constraint) error {
	dgraphConstraint, err := dgraphConstraintFromPb(constraint)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(dgraphConstraint)
	if err != nil {
		return err
	}

	mu := &api.Mutation{
		CommitNow: true,
	}

	mu.SetJson = payload

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, err = d.dgraphClient.NewTxn().Mutate(ctx, mu)
	if err != nil {
		return err
	}

	return nil
}

// dgraphConstraintFromPb converts a protobuf Constraint to a Dgraph Constraint.
//
// Parameters:
// - pbConstraint: The protobuf Constraint message to convert.
//
// Returns:
// - A Dgraph Constraint struct.
// - An error if there was an issue converting the constraint.
func dgraphConstraintFromPb(pbConstraint *tarianpb.Constraint) (*Constraint, error) {
	selectorJSON, err := json.Marshal(pbConstraint.GetSelector())
	if err != nil {
		return nil, err
	}

	allowedProcessesJSON, err := json.Marshal(pbConstraint.GetAllowedProcesses())
	if err != nil {
		return nil, err
	}

	allowedFilesJSON, err := json.Marshal(pbConstraint.GetAllowedFiles())
	if err != nil {
		return nil, err
	}

	dgraphConstraint := Constraint{
		UID:              "_:constraint",
		DType:            []string{"Constraint"},
		Namespace:        pbConstraint.GetNamespace(),
		Name:             pbConstraint.GetName(),
		Selector:         string(selectorJSON),
		AllowedProcesses: string(allowedProcessesJSON),
		AllowedFiles:     string(allowedFilesJSON),
	}

	return &dgraphConstraint, nil
}

// RemoveByNamespaceAndName removes a constraint by namespace and name from the Dgraph store.
//
// Parameters:
// - namespace: The namespace of the constraint to remove.
// - name: The name of the constraint to remove.
//
// Returns:
// - An error if there was an issue removing the constraint from the database.
func (d *DgraphConstraintStore) RemoveByNamespaceAndName(namespace, name string) error {
	uid, err := d.findConstraintUIDByNamespaceAndName(namespace, name)

	if err != nil {
		return err
	}

	if uid == "" {
		return nil
	}

	q := fmt.Sprintf(`{"uid": "%s"}`, uid)

	op := &api.Mutation{DeleteJson: []byte(q), CommitNow: true}
	txn := d.dgraphClient.NewTxn()

	ctxDiscard, cancelDiscard := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelDiscard()
	defer func() {
		_ = txn.Discard(ctxDiscard)
	}()

	ctxMutate, cancelMutate := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelMutate()
	_, err = txn.Mutate(ctxMutate, op)

	return err
}

// findConstraintUIDByNamespaceAndName searches for the UID of a constraint by namespace and name.
//
// Parameters:
// - namespace: The namespace of the constraint to find.
// - name: The name of the constraint to find.
//
// Returns:
// - The UID of the constraint, or an empty string if not found.
// - An error if there was an issue with the database query.
func (d *DgraphConstraintStore) findConstraintUIDByNamespaceAndName(namespace, name string) (string, error) {
	const q = `
		query constraintQuery($namespace: string, $name: string) {
			constraints(func: type(Constraint), first: 1) @filter(eq(constraint_namespace, $namespace) AND eq(constraint_name, $name)) {
				uid
			}
		}
	`

	tx := d.dgraphClient.NewReadOnlyTxn()
	vars := map[string]string{"$namespace": namespace, "$name": name}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := tx.QueryWithVars(ctx, q, vars)
	if err != nil {
		return "", err
	}

	var result dgraphConstraintList
	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return "", err
	}

	if len(result.Constraints) == 0 {
		return "", nil
	}

	return result.Constraints[0].UID, nil
}
