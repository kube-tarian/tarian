package dbstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/driftprogramming/pgxpoolmock"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// DbEventStore implements store.EventStore
type DbEventStore struct {
	// pool holds the connection pool. It uses a compatible interface with the real PgxPool.
	// This is to make it mockable.
	pool pgxpoolmock.PgxPool
}

func NewDbEventStore(dsn string) (*DbEventStore, error) {
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

	store := &DbEventStore{pool: dbpool}

	return store, nil
}

// eventRow represents a row of database table events
// Fields are exported because to work around type
// being a reserved name.
type eventRow struct {
	ID              int
	UID             string
	Type            string
	ServerTimestamp time.Time
	ClientTimestamp time.Time
	AlertSentAt     sql.NullTime
	Targets         string
}

func (e *eventRow) toEvent() *tarianpb.Event {
	event := tarianpb.NewEvent()
	event.Uid = e.UID
	event.Type = e.Type
	event.ServerTimestamp = timestamppb.New(e.ServerTimestamp)
	event.ClientTimestamp = timestamppb.New(e.ClientTimestamp)

	if e.AlertSentAt.Valid {
		event.AlertSentAt = timestamppb.New(e.AlertSentAt.Time)
	}

	json.Unmarshal([]byte(e.Targets), &event.Targets)

	return event
}

func (d *DbEventStore) GetAll(limit uint) ([]*tarianpb.Event, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM events ORDER BY id ASC LIMIT $1", limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPbEvents(rows)
}

func (d *DbEventStore) FindByNamespace(namespace string, limit uint) ([]*tarianpb.Event, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM events WHERE namespace = $1 ORDER BY id ASC LIMIT $2", namespace, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPbEvents(rows)
}

func (d *DbEventStore) FindWhereAlertNotSent() ([]*tarianpb.Event, error) {
	oneDayAgo := time.Now().UTC().AddDate(0, 0, -1)

	rows, err := d.pool.Query(context.Background(), "SELECT * FROM events WHERE server_timestamp > $1 AND alert_sent_at IS NULL ORDER BY id ASC", oneDayAgo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPbEvents(rows)
}

func rowsToPbEvents(rows pgx.Rows) ([]*tarianpb.Event, error) {
	events := []*tarianpb.Event{}

	for rows.Next() {
		e := eventRow{}

		err := rows.Scan(&e.ID, &e.UID, &e.Type, &e.ServerTimestamp, &e.ClientTimestamp, &e.AlertSentAt, &e.Targets)
		if err != nil {
			return nil, err
		}

		events = append(events, e.toEvent())
	}

	return events, nil
}

func (d *DbEventStore) Add(event *tarianpb.Event) error {
	var id int
	targetsJSON, err := json.Marshal(event.GetTargets())
	if err != nil {
		return err
	}

	uid := uuid.NewV4()

	err = d.pool.
		QueryRow(
			context.Background(),
			"INSERT INTO events(uid, type, server_timestamp, client_timestamp, targets) VALUES($1, $2, $3, $4, $5) RETURNING id",
			uid.String(), event.GetType(), event.GetServerTimestamp().AsTime(), event.GetClientTimestamp().AsTime(), targetsJSON).
		Scan(&id)
	if err != nil {
		return err
	}

	return nil
}

func (d *DbEventStore) UpdateAlertSent(uid string) error {
	_, err := d.pool.Exec(
		context.Background(),
		"UPDATE events SET alert_sent_at = $1 WHERE uid = $2",
		time.Now().UTC(), uid)

	return err
}
