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
		err = tx.QueryRowContext(ctx, "SELECT amount FROM ledger_entries WHERE id = $1", entry.ID).Scan(&amount)
		assert.NoError(t, err)
		assert.Equal(t, int64(200), amount)
	})

	t.Run("insert conflict do nothing", func(t *testing.T) {
		// Attempting to insert duplicate should succeed without error (DO NOTHING)
		err := repo.SaveLedgerEntry(ctx, entry)
		assert.NoError(t, err)
	})
}

func TestSumLedgerByUserIDWithCutOffTime(t *testing.T) {
	tx := beginTx(t)
	repo := postgres.NewPostgresLedgerRepoWithTransaction(tx)
	ctx := t.Context()

	now := time.Now().UTC().Round(time.Microsecond)
	cutoffTime := now.Add(-10 * time.Minute)

	t.Run("success with matching entries and exclusions", func(t *testing.T) {
		// Entry 1: target user, after cutoff
		entry1 := &domain.LedgerEntry{
			ID:             "ledger-sum-1",
			UserID:         "user-sum",
			IdempotencyKey: "idem-1",
			TransactionID:  "tx-1",
			Amount:         100,
			Type:           domain.LedgerEntryTypeTipTransfer,
			CreatedAt:      cutoffTime.Add(2 * time.Minute),
		}
		// Entry 2: target user, after cutoff
		entry2 := &domain.LedgerEntry{
			ID:             "ledger-sum-2",
			UserID:         "user-sum",
			IdempotencyKey: "idem-2",
			TransactionID:  "tx-2",
			Amount:         250,
			Type:           domain.LedgerEntryTypeTipTransfer,
			CreatedAt:      cutoffTime.Add(5 * time.Minute),
		}
		// Entry 3: target user, before cutoff (should NOT be included)
		entry3 := &domain.LedgerEntry{
			ID:             "ledger-sum-3",
			UserID:         "user-sum",
			IdempotencyKey: "idem-3",
			TransactionID:  "tx-3",
			Amount:         500,
			Type:           domain.LedgerEntryTypeTipTransfer,
			CreatedAt:      cutoffTime.Add(-5 * time.Minute),
		}
		// Entry 4: different user, after cutoff (should NOT be included)
		entry4 := &domain.LedgerEntry{
			ID:             "ledger-sum-4",
			UserID:         "other-user",
			IdempotencyKey: "idem-4",
			TransactionID:  "tx-4",
			Amount:         300,
			Type:           domain.LedgerEntryTypeTipTransfer,
			CreatedAt:      cutoffTime.Add(3 * time.Minute),
		}
		// Entry 5: target user, exactly at cutoff time (created_at > cutoffTime -> excluded)
		entry5 := &domain.LedgerEntry{
			ID:             "ledger-sum-5",
			UserID:         "user-sum",
			IdempotencyKey: "idem-5",
			TransactionID:  "tx-5",
			Amount:         400,
			Type:           domain.LedgerEntryTypeTipTransfer,
			CreatedAt:      cutoffTime,
		}

		for _, e := range []*domain.LedgerEntry{entry1, entry2, entry3, entry4, entry5} {
			err := repo.SaveLedgerEntry(ctx, e)
			assert.NoError(t, err)
		}

		sum, err := repo.SumLedgerByUserIDWithCutOffTime(ctx, "user-sum", cutoffTime)
		assert.NoError(t, err)
		assert.Equal(t, int64(350), sum)
	})

	t.Run("no entries matching", func(t *testing.T) {
		sum, err := repo.SumLedgerByUserIDWithCutOffTime(ctx, "non-existent-user", cutoffTime)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), sum)
	})
}
