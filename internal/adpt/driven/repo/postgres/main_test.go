package postgres_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	embeddedpg "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/st-ember/microtip/internal/adpt/driven/repo/postgres"
)

var TestDB *sql.DB
var dbURL = "postgres://postgres:postgres@localhost:5436/postgres?sslmode=disable"

func TestMain(m *testing.M) {
	config := embeddedpg.DefaultConfig().
		Port(5436).
		Logger(nil)

	postgresDb := embeddedpg.NewDatabase(config)
	if err := postgresDb.Start(); err != nil {
		log.Fatalf("start embedded postgres: %v", err)
	}

	var err error
	TestDB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("connect to embedded postgres: %v", err)
	}

	// Create database schema matching our domain
	createTablesSQL := `
        CREATE TABLE IF NOT EXISTS balance (
            user_id TEXT PRIMARY KEY,
            current_balance BIGINT,
            created_at TIMESTAMPTZ,
            updated_at TIMESTAMPTZ
        );
        CREATE TABLE IF NOT EXISTS ledger_entry (
            id TEXT PRIMARY KEY,
            user_id TEXT,
            idempotency_key TEXT,
            transaction_id TEXT,
            amount INT,
            type TEXT,
            created_at TIMESTAMPTZ
        );
	`
	_, err = TestDB.Exec(createTablesSQL)
	if err != nil {
		log.Fatalf("create tables in embedded postgres: %v", err)
	}

	code := m.Run()

	if err := TestDB.Close(); err != nil {
		log.Fatalf("close test db connection: %v", err)
	}

	if err := postgresDb.Stop(); err != nil {
		log.Fatalf("stop embedded postgres: %v", err)
	}

	os.Exit(code)
}

func beginTx(t *testing.T) *sql.Tx {
	tx, err := TestDB.BeginTx(t.Context(), nil)
	require.NoError(t, err)

	_, err = tx.ExecContext(t.Context(), "TRUNCATE balance, ledger_entry RESTART IDENTITY CASCADE;")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return tx
}

func TestNewDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	t.Run("success", func(t *testing.T) {
		db, err := postgres.NewDB(ctx, dbURL)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		defer func() {
			_ = db.Conn.Close()
		}()
	})

	t.Run("invalid connection url", func(t *testing.T) {
		// This should fail to connect/ping
		db, err := postgres.NewDB(ctx, "postgres://postgres:postgres@localhost:9999/invalid_db?sslmode=disable")
		assert.Error(t, err)
		assert.Nil(t, db)
	})
}
