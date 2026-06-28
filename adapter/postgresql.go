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
	if err := a.db.PingContext(ctx); err != nil {
		a.db.Close()
		return err
	}
	if err := a.ensureInitialized(ctx); err != nil {
		a.db.Close()
		return err
	}
	return nil
}

func (a *postgresAdapter) ensureInitialized(ctx context.Context) error {
	if _, err := a.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS keepalive (
	key TEXT PRIMARY KEY,
	value BIGINT NOT NULL DEFAULT 0
)`); err != nil {
		return err
	}
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO keepalive (key, value) VALUES ($1, 0) ON CONFLICT (key) DO NOTHING`,
		a.key,
	)
	return err
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
	if a.db == nil {
		return nil
	}
	return a.db.Close()
}
