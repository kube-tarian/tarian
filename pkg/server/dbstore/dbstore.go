package dbstore

import (
	"context"
	"embed"
	"encoding/json"

	"github.com/Boostport/migration"
	"github.com/Boostport/migration/driver/postgres"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/driftprogramming/pgxpoolmock"

	"github.com/jackc/pgx/v4/pgxpool"
)

//go:embed migrations
var embedFS embed.FS

func MigrationSource() *migration.EmbedMigrationSource {
	return &migration.EmbedMigrationSource{
		EmbedFS: embedFS,
		Dir:     "migrations",
	}
}

func RunMigration(dsn string) (int, error) {
	driver, err := postgres.New(dsn)

	if err != nil {
		return 0, err
	}

	applied, err := migration.Migrate(driver, MigrationSource(), migration.Up, 0)

	return applied, err
}

// DbConstraintStore implements store.ConstraintStore
type DbConstraintStore struct {
	// pool holds the connection pool. It uses a compatible interface with the real PgxPool.
	// This is to make it mockable.
	pool pgxpoolmock.PgxPool
}

func NewDbConstraintStore(dsn string) (*DbConstraintStore, error) {
	// TODO: pass context from param?
	dbpool, err := pgxpool.Connect(context.Background(), dsn)

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
	selector         string
	allowedProcesses string
}

func (c *constraintRow) toConstraint() *tarianpb.Constraint {
	constraint := &tarianpb.Constraint{}
	constraint.Namespace = c.namespace
	json.Unmarshal([]byte(c.selector), &constraint.Selector)
	json.Unmarshal([]byte(c.allowedProcesses), &constraint.AllowedProcesses)

	return constraint
}

func (d *DbConstraintStore) GetAll() ([]*tarianpb.Constraint, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM constraints")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	allConstraints := []*tarianpb.Constraint{}

	for rows.Next() {
		r := constraintRow{}

		err := rows.Scan(&r.id, &r.namespace, &r.selector, &r.allowedProcesses)
		if err != nil {
			// TODO: logger.Errorw()

			continue
		}

		constraint := r.toConstraint()

		allConstraints = append(allConstraints, constraint)
	}

	return allConstraints, nil
}

func (d *DbConstraintStore) FindByNamespace(namespace string) ([]*tarianpb.Constraint, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM constraints WHERE namespace = $1", namespace)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	constraints := []*tarianpb.Constraint{}

	for rows.Next() {
		r := constraintRow{}

		err := rows.Scan(&r.id, &r.namespace, &r.selector, &r.allowedProcesses)
		if err != nil {
			// TODO: logger.Errorw()

			continue
		}

		constraint := r.toConstraint()

		constraints = append(constraints, constraint)
	}

	return constraints, nil
}

func (d *DbConstraintStore) Add(constraint *tarianpb.Constraint) error {
	var id int
	selectorJson, err := json.Marshal(constraint.GetSelector())
	if err != nil {
		return err
	}

	allowedProcessesJson, err := json.Marshal(constraint.GetAllowedProcesses())
	if err != nil {
		return err
	}

	err = d.pool.
		QueryRow(
			context.Background(),
			"INSERT INTO constraints(namespace, selector, allowed_processes) VALUES($1, $2, $3) RETURNING id",
			constraint.GetNamespace(), selectorJson, allowedProcessesJson).
		Scan(&id)
	if err != nil {
		return err
	}

	return nil
}
