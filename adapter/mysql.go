package adapter

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	Registry["mysql"] = func() (Adapter, error) { return &mysqlAdapter{}, nil }
}

type mysqlAdapter struct {
	db *sql.DB
}

func (a *mysqlAdapter) Connect(ctx context.Context) error {
	dsn, err := envOrFail("DATA_SOURCE_NAME")
	if err != nil {
		return err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	a.db = db
	return a.db.PingContext(ctx)
}

func (a *mysqlAdapter) Increment(ctx context.Context) (int64, error) {
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE `keepalive` SET `value` = `value` + 1 WHERE `key` = 'counter'",
	); err != nil {
		tx.Rollback()
		return 0, err
	}
	var value int64
	if err := tx.QueryRowContext(ctx,
		"SELECT `value` FROM `keepalive` WHERE `key` = 'counter'",
	).Scan(&value); err != nil {
		tx.Rollback()
		return 0, err
	}
	return value, tx.Commit()
}

func (a *mysqlAdapter) Close(_ context.Context) error {
	return a.db.Close()
}
