package adapter

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
)

func init() {
	Registry["postgresql"] = func() (Adapter, error) { return &postgresAdapter{}, nil }
	Registry["postgres"] = Registry["postgresql"]
}

type postgresAdapter struct {
	db  *sql.DB
	key string
}

func (a *postgresAdapter) Connect(ctx context.Context) error {
	uri, err := envOrFail("KEEPALIVE_POSTGRESQL_URL")
	if err != nil {
		return err
	}
	db, err := sql.Open("postgres", uri)
	if err != nil {
		return err
	}
	a.db = db
	a.key = envOr("KEEPALIVE_COUNTER_KEY", "counter")
	return a.db.PingContext(ctx)
}

func (a *postgresAdapter) Increment(ctx context.Context) (int64, error) {
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	var value int64
	if err := tx.QueryRowContext(ctx,
		`UPDATE keepalive SET value = value + 1 WHERE key = $1 RETURNING value`,
		a.key,
	).Scan(&value); err != nil {
		tx.Rollback()
		return 0, err
	}
	return value, tx.Commit()
}

func (a *postgresAdapter) Close(_ context.Context) error {
	return a.db.Close()
}
