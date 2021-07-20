package dbstore

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/Boostport/migration"
	"github.com/Boostport/migration/driver/postgres"
	"github.com/devopstoday11/tarian/pkg/tarianpb"

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
	pool *pgxpool.Pool
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

func (d *DbConstraintStore) GetAll() ([]*tarianpb.Constraint, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM constraints")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	allConstraints := []*tarianpb.Constraint{}

	for rows.Next() {
		c := &tarianpb.Constraint{}
		rows.Scan(c.Namespace)

		var selectorJson string
		var allowedProcessesJson string

		rows.Scan(selectorJson)
		rows.Scan(allowedProcessesJson)

		json.Unmarshal([]byte(selectorJson), c.Selector)
		json.Unmarshal([]byte(allowedProcessesJson), &c.AllowedProcesses)

		fmt.Println("test")
	}

	return allConstraints, nil
}

func (d *DbConstraintStore) FindByNamespace(namespace string) ([]*tarianpb.Constraint, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM constraints WHERE namespace = $1", namespace)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	allConstraints := []*tarianpb.Constraint{}

	for rows.Next() {
		constraint := &tarianpb.Constraint{}

		var id int
		var selectorJson string
		var allowedProcessesJson string
		err := rows.Scan(&id, &constraint.Namespace, &selectorJson, &allowedProcessesJson)
		if err != nil {
			fmt.Println(err)
		}

		json.Unmarshal([]byte(selectorJson), &constraint.Selector)
		json.Unmarshal([]byte(allowedProcessesJson), &constraint.AllowedProcesses)

		allConstraints = append(allConstraints, constraint)
	}

	return allConstraints, nil
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
