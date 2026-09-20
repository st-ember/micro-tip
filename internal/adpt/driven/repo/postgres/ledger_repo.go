package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/st-ember/microtip/internal/app/port/repo"
	"github.com/st-ember/microtip/internal/domain"
)

type PostgresLedgerRepo struct {
	tx *sql.Tx
}

func NewPostgresLedgerRepoWithTransaction(tx *sql.Tx) repo.LedgerRepo {
	return &PostgresLedgerRepo{tx}
}

func (lr *PostgresLedgerRepo) SaveLedgerEntry(ctx context.Context, entry *domain.LedgerEntry) error {
	query := `
		INSERT INTO ledger_entry (id, user_id, idempotency_key, transaction_id, amount, type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := lr.tx.ExecContext(
		ctx, query, entry.ID, entry.UserID, entry.IdempotencyKey,
		entry.TransactionID, entry.Amount, entry.Type, entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save ledger entry: %w", err)
	}

	return nil
}
