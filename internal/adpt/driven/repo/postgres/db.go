package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DB struct {
	Conn *sql.DB
}

func NewDB(ctx context.Context, dbURL string) (*DB, error) {
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres database: %w", err)
	}

	return &DB{conn}, nil
}
