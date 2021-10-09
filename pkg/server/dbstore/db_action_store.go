package dbstore

import (
	"context"
	"encoding/json"

	"github.com/driftprogramming/pgxpoolmock"
	"github.com/kube-tarian/tarian/pkg/tarianpb"

	"github.com/jackc/pgx/v4/pgxpool"
)

// DbActionStore implements store.ActionStore
type DbActionStore struct {
	// pool holds the connection pool. It uses a compatible interface with the real PgxPool.
	// This is to make it mockable.
	pool pgxpoolmock.PgxPool
}

func NewDbActionStore(dsn string) (*DbActionStore, error) {
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

	store := &DbActionStore{pool: dbpool}

	return store, nil
}

// constraintRow represents a row of database table actions
type actionRow struct {
	id                 int
	namespace          string
	name               string
	selector           string
	onViolatedProcess  bool
	onViolatedFile     bool
	onFalcoAlert       bool
	falcoAlertPriority int32
	action             string
}

func (a *actionRow) toAction() *tarianpb.Action {
	action := tarianpb.NewAction()
	action.Namespace = a.namespace
	action.Name = a.name
	json.Unmarshal([]byte(a.selector), &action.Selector)
	action.OnViolatedProcess = a.onViolatedProcess
	action.OnViolatedFile = a.onViolatedFile
	action.OnFalcoAlert = a.onFalcoAlert

	action.FalcoPriority = tarianpb.FalcoPriority(a.falcoAlertPriority)
	action.Action = a.action

	return action
}

func (d *DbActionStore) GetAll() ([]*tarianpb.Action, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM actions ORDER BY id ASC")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	allActions := []*tarianpb.Action{}

	for rows.Next() {
		r := actionRow{}

		err := rows.Scan(&r.id, &r.namespace, &r.name, &r.selector, &r.onViolatedProcess, &r.onViolatedFile, &r.onFalcoAlert, &r.falcoAlertPriority, &r.action)
		if err != nil {
			return nil, err
		}

		action := r.toAction()

		allActions = append(allActions, action)
	}

	return allActions, nil
}

func (d *DbActionStore) FindByNamespace(namespace string) ([]*tarianpb.Action, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM actions WHERE namespace = $1 ORDER BY id ASC", namespace)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	actions := []*tarianpb.Action{}

	for rows.Next() {
		r := actionRow{}

		err := rows.Scan(&r.id, &r.namespace, &r.name, &r.selector, &r.onViolatedProcess, &r.onViolatedFile, &r.onFalcoAlert, &r.falcoAlertPriority, &r.action)
		if err != nil {
			return nil, err
		}

		action := r.toAction()

		actions = append(actions, action)
	}

	return actions, nil
}

func (d *DbActionStore) Add(action *tarianpb.Action) error {
	var id int
	selectorJSON, err := json.Marshal(action.GetSelector())
	if err != nil {
		return err
	}

	err = d.pool.
		QueryRow(
			context.Background(),
			"INSERT INTO actions(namespace, name, selector, on_violated_process, on_violated_file, on_falco_alert, falco_alert_priority, action) VALUES($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id",
			action.GetNamespace(), action.GetName(), selectorJSON, action.GetOnViolatedProcess(), action.GetOnViolatedFile(), action.GetOnFalcoAlert(), action.GetFalcoPriority(), action.GetAction()).
		Scan(&id)
	if err != nil {
		return err
	}

	return nil
}

func (d *DbActionStore) NamespaceAndNameExist(namespace, name string) (bool, error) {
	exist := false

	rows, err := d.pool.Query(context.Background(), "SELECT 1 FROM actions WHERE namespace = $1 AND name = $2 LIMIT 1", namespace, name)
	if err != nil {
		return false, err
	}

	defer rows.Close()

	if rows.Next() {
		exist = true
	}

	return exist, nil
}

func (d *DbActionStore) RemoveByNamespaceAndName(namespace, name string) error {
	_, err := d.pool.
		Exec(
			context.Background(),
			"DELETE FROM actions WHERE namespace = $1 AND name = $2",
			namespace, name,
		)

	return err
}
