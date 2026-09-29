package db

import (
	"context"
	"database/sql"
	_ "embed"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultDSN = "postgres://jobsuser:jobspass@localhost:5432/jobsdb?sslmode=disable"

//go:embed schema.sql
var schema string

// Connect opens a connection pool to PostgreSQL and verifies it with a ping.
// The DSN is read from DATABASE_URL, falling back to the compose.yml defaults.
func Connect(ctx context.Context) (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = defaultDSN
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// Migrate creates the tables if they do not already exist.
func Migrate(ctx context.Context, conn *sql.DB) error {
	_, err := conn.ExecContext(ctx, schema)
	return err
}
