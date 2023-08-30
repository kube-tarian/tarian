package dgraphstore

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"golang.org/x/net/context"
)

type DgraphConstraintStore struct {
	dgraphClient *dgo.Dgraph
}

func NewDgraphConstraintStore(dgraphClient *dgo.Dgraph) *DgraphConstraintStore {
	d := &DgraphConstraintStore{dgraphClient: dgraphClient}

	return d
}

func (d *DgraphConstraintStore) GetAll() ([]*tarianpb.Constraint, error) {
	q := fmt.Sprintf(`
		{
			constraints(func: type(Constraint)) {
				%s
			}
		}
	`, constraintFields)

	tx := d.dgraphClient.NewReadOnlyTxn()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

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

const constraintFields = `
	uid
	dgraph.type

	constraint_namespace
	constraint_name
	constraint_selector
	constraint_allowed_processes
	constraint_allowed_files
`

type dgraphConstraintList struct {
	Constraints []Constraint
}

func (d *dgraphConstraintList) toPbConstraints() []*tarianpb.Constraint {
	constraints := []*tarianpb.Constraint{}
	for _, c := range d.Constraints {
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

func (d *DgraphConstraintStore) FindByNamespace(namespace string) ([]*tarianpb.Constraint, error) {
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

func (d *DgraphConstraintStore) NamespaceAndNameExist(namespace, name string) (bool, error) {
	uid, err := d.findConstraintUIDByNamespaceAndName(namespace, name)

	if err != nil {
		return false, err
	}

	exist := uid != ""
	return exist, err
}

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
