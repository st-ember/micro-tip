package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/st-ember/microtip/internal/domain"
)

func TestNewLedgerEntry(t *testing.T) {
	const (
		id     = "entry-123"
		userID = "user-123"
		idpKey = "idp-123"
		txID   = "tx-123"
	)

	t.Run("NewLedgerEntry - successful creation", func(t *testing.T) {
		beforeCreation := time.Now()

		entry, err := domain.NewLedgerEntry(id, userID, idpKey, txID, 100, domain.LedgerEntryTypeTopUp)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, id, entry.ID)
		assert.Equal(t, userID, entry.UserID)
		assert.Equal(t, idpKey, entry.IdempotencyKey)
		assert.Equal(t, txID, entry.TransactionID)
		assert.Equal(t, int64(100), entry.Amount)
		assert.Equal(t, domain.LedgerEntryType(domain.LedgerEntryTypeTopUp), entry.Type)
		assert.True(t, entry.CreatedAt.After(beforeCreation) || entry.CreatedAt.Equal(beforeCreation))
	})

	t.Run("NewLedgerEntry - error on zero amount", func(t *testing.T) {
		entry, err := domain.NewLedgerEntry(id, userID, idpKey, txID, 0, domain.LedgerEntryTypeTopUp)
		assert.Nil(t, entry)
		assert.ErrorIs(t, err, domain.ErrInvalidLedgerAmt)
	})

	t.Run("NewLedgerEntry - error on invalid entry type", func(t *testing.T) {
		entry, err := domain.NewLedgerEntry(id, userID, idpKey, txID, 100, domain.LedgerEntryType("INVALID"))
		assert.Nil(t, entry)
		assert.ErrorIs(t, err, domain.ErrEntryTypeInvalid)
	})
}
