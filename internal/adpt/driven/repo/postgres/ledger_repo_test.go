package postgres_test

import (
	"testing"
	"time"

	"github.com/st-ember/microtip/internal/adpt/driven/repo/postgres"
	"github.com/st-ember/microtip/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestSaveLedgerEntry(t *testing.T) {
	tx := beginTx(t)
	repo := postgres.NewPostgresLedgerRepoWithTransaction(tx)
	ctx := t.Context()

	entry := &domain.LedgerEntry{
		ID:             "ledger-1",
		UserID:         "user-1",
		IdempotencyKey: "idem-key-123",
		TransactionID:  "tx-123",
		Amount:         200,
		Type:           domain.LedgerEntryTypeTopUp,
		CreatedAt:      time.Now().UTC().Round(time.Microsecond),
	}

	t.Run("insert success", func(t *testing.T) {
		err := repo.SaveLedgerEntry(ctx, entry)
		assert.NoError(t, err)

		var amount int64
		err = tx.QueryRowContext(ctx, "SELECT amount FROM ledger_entry WHERE id = $1", entry.ID).Scan(&amount)
		assert.NoError(t, err)
		assert.Equal(t, int64(200), amount)
	})

	t.Run("insert conflict do nothing", func(t *testing.T) {
		// Attempting to insert duplicate should succeed without error (DO NOTHING)
		err := repo.SaveLedgerEntry(ctx, entry)
		assert.NoError(t, err)
	})
}
