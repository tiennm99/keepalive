package adapter

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
)

func init() {
	Registry["postgresql"] = func(cfg Config) (Adapter, error) {
		url, err := cfg.Required("url")
		if err != nil {
			return nil, err
		}
		return &postgresAdapter{
			url: url,
			key: cfg.Optional("counter_key", "counter"),
		}, nil
	}
	Registry["postgres"] = Registry["postgresql"]
}

type postgresAdapter struct {
	db  *sql.DB
	url string
	key string
}

func (a *postgresAdapter) Connect(ctx context.Context) error {
	db, err := sql.Open("postgres", a.url)
	if err != nil {
		return err
	}
	a.db = db
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
