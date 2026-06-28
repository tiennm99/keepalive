package adapter

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	Registry["mysql"] = func(cfg Config) (Adapter, error) {
		dsn, err := cfg.Required("dsn")
		if err != nil {
			return nil, err
		}
		return &mysqlAdapter{
			dsn: dsn,
			key: cfg.Optional("counter_key", "counter"),
		}, nil
	}
}

type mysqlAdapter struct {
	db  *sql.DB
	dsn string
	key string
}

func (a *mysqlAdapter) Connect(ctx context.Context) error {
	db, err := sql.Open("mysql", a.dsn)
	if err != nil {
		return err
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
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

func (a *mysqlAdapter) ensureInitialized(ctx context.Context) error {
	if _, err := a.db.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS `keepalive` (`key` VARCHAR(255) PRIMARY KEY, `value` BIGINT NOT NULL DEFAULT 0)",
	); err != nil {
		return err
	}
	_, err := a.db.ExecContext(ctx,
		"INSERT IGNORE INTO `keepalive` (`key`, `value`) VALUES (?, 0)",
		a.key,
	)
	return err
}

func (a *mysqlAdapter) Increment(ctx context.Context) (int64, error) {
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE `keepalive` SET `value` = `value` + 1 WHERE `key` = ?",
		a.key,
	); err != nil {
		tx.Rollback()
		return 0, err
	}
	var value int64
	if err := tx.QueryRowContext(ctx,
		"SELECT `value` FROM `keepalive` WHERE `key` = ?",
		a.key,
	).Scan(&value); err != nil {
		tx.Rollback()
		return 0, err
	}
	return value, tx.Commit()
}

func (a *mysqlAdapter) Close(_ context.Context) error {
	if a.db == nil {
		return nil
	}
	return a.db.Close()
}
