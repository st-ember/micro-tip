package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/st-ember/microtip/internal/app/port/repo"
)

// PostgresUnitOfWork implements the UnitOfWork interface for PostgreSQL
// and holds the transaction object.
type PostgresUnitOfWork struct {
	tx        *sql.Tx
	committed bool
}

// PostgresUnitOfWorkFactory is the factory for creating UoW instances
// and holds the connection pool
type PostgresUnitOfWorkFactory struct {
	db *sql.DB
}

// NewPostgresUnitOfWorkFactory creates a new factory instance.
func NewPostgresUnitOfWorkFactory(db *sql.DB) repo.UnitOfWorkFactory {
	return &PostgresUnitOfWorkFactory{db}
}

// NewUnitOfWork is called every time a database transaction is needed
func (f *PostgresUnitOfWorkFactory) NewUnitOfWork(ctx context.Context) (repo.UnitOfWork, error) {
	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("start new transaction: %w", err)
	}

	return &PostgresUnitOfWork{tx: tx, committed: false}, nil
}

func (u *PostgresUnitOfWork) BalanceRepo() repo.BalanceRepo {
	return NewPostgresBalanceRepoWithTransaction(u.tx)
}

func (u *PostgresUnitOfWork) LedgerRepo() repo.LedgerRepo {
	return NewPostgresLedgerRepoWithTransaction(u.tx)
}

func (u *PostgresUnitOfWork) Commit(ctx context.Context) error {
	if u.committed {
		return errors.New("transaction already committed")
	}

	u.committed = true

	return u.tx.Commit()
}

func (u *PostgresUnitOfWork) Rollback(ctx context.Context) error {
	if u.committed {
		return nil // ignore
	}

	u.committed = true

	return u.tx.Rollback()
}

func (u *PostgresUnitOfWork) Close(ctx context.Context) error {
	return u.Rollback(ctx)
}
