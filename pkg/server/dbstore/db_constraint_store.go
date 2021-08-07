package dbstore

import (
	"context"
	"encoding/json"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/driftprogramming/pgxpoolmock"

	"github.com/jackc/pgx/v4/pgxpool"
)

// DbConstraintStore implements store.ConstraintStore
type DbConstraintStore struct {
	// pool holds the connection pool. It uses a compatible interface with the real PgxPool.
	// This is to make it mockable.
	pool pgxpoolmock.PgxPool
}

func NewDbConstraintStore(dsn string) (*DbConstraintStore, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	poolConfig.LazyConnect = true

	// ctx is used to connect initially if it's not lazyConnect, which
	// is not our case here. So, it's ok to use context.Background().
	dbpool, err := pgxpool.ConnectConfig(context.Background(), poolConfig)

	if err != nil {
		return nil, err
	}

	store := &DbConstraintStore{pool: dbpool}

	return store, nil
}

// constraintRow represents a row of database table constraints
type constraintRow struct {
	id               int
	namespace        string
	name             string
	selector         string
	allowedProcesses string
}

func (c *constraintRow) toConstraint() *tarianpb.Constraint {
	constraint := tarianpb.NewConstraint()
	constraint.Namespace = c.namespace
	constraint.Name = c.name
	json.Unmarshal([]byte(c.selector), &constraint.Selector)
	json.Unmarshal([]byte(c.allowedProcesses), &constraint.AllowedProcesses)

	return constraint
}

func (d *DbConstraintStore) GetAll() ([]*tarianpb.Constraint, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM constraints ORDER BY id ASC")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	allConstraints := []*tarianpb.Constraint{}

	for rows.Next() {
		r := constraintRow{}

		err := rows.Scan(&r.id, &r.namespace, &r.name, &r.selector, &r.allowedProcesses)
		if err != nil {
			return nil, err
		}

		constraint := r.toConstraint()

		allConstraints = append(allConstraints, constraint)
	}

	return allConstraints, nil
}

func (d *DbConstraintStore) FindByNamespace(namespace string) ([]*tarianpb.Constraint, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM constraints WHERE namespace = $1 ORDER BY id ASC", namespace)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	constraints := []*tarianpb.Constraint{}

	for rows.Next() {
		r := constraintRow{}

		err := rows.Scan(&r.id, &r.namespace, &r.name, &r.selector, &r.allowedProcesses)
		if err != nil {
			return nil, err
		}

		constraint := r.toConstraint()

		constraints = append(constraints, constraint)
	}

	return constraints, nil
}

func (d *DbConstraintStore) Add(constraint *tarianpb.Constraint) error {
	var id int
	selectorJSON, err := json.Marshal(constraint.GetSelector())
	if err != nil {
		return err
	}

	allowedProcessesJSON, err := json.Marshal(constraint.GetAllowedProcesses())
	if err != nil {
		return err
	}

	err = d.pool.
		QueryRow(
			context.Background(),
			"INSERT INTO constraints(namespace, name, selector, allowed_processes) VALUES($1, $2, $3, $4) RETURNING id",
			constraint.GetNamespace(), constraint.GetName(), selectorJSON, allowedProcessesJSON).
		Scan(&id)
	if err != nil {
		return err
	}

	return nil
}

func (d *DbConstraintStore) NamespaceAndNameExist(namespace, name string) (bool, error) {
	exist := false

	rows, err := d.pool.Query(context.Background(), "SELECT 1 FROM constraints WHERE namespace = $1 AND name = $2 LIMIT 1", namespace, name)
	if err != nil {
		return false, err
	}

	defer rows.Close()

	if rows.Next() {
		exist = true
	}

	return exist, nil
}
