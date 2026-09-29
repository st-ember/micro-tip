package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
		_ = conn.Close()
		return nil, fmt.Errorf("ping postgres database: %w", err)
	}

	// Optimize connection pool limits for high concurrency and spike tests
	conn.SetMaxOpenConns(50)                  // Cap maximum open connections to prevent DB exhaustion
	conn.SetMaxIdleConns(25)                  // Keep warm idle connections to eliminate connection churn
	conn.SetConnMaxLifetime(10 * time.Minute) // Retire old connections safely
	conn.SetConnMaxIdleTime(5 * time.Minute)  // Clean up idle connections to free DB resources

	return &DB{conn}, nil
}
