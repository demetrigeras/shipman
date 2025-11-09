package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var Pool *sql.DB

func Open(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(15)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(2 * time.Hour)

	return conn, nil
}

func SetPool(db *sql.DB) {
	Pool = db
}

func Ping(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
