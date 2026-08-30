package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/st-ember/microtip/internal/domain"
)

func TestNewBalance(t *testing.T) {
	t.Run("NewBalance - initializes fields correctly", func(t *testing.T) {
		userID := "user-123"
		startingBalance := int64(500)
		beforeCreation := time.Now()

		b := domain.NewBalance(userID, startingBalance)

		assert.Equal(t, userID, b.UserID)
		assert.Equal(t, startingBalance, b.CurrentBalance)
		assert.True(t, b.CreatedAt.After(beforeCreation) || b.CreatedAt.Equal(beforeCreation))
		assert.True(t, b.UpdatedAt.After(beforeCreation) || b.UpdatedAt.Equal(beforeCreation))
	})
}

func TestBalance_UpdateBalance(t *testing.T) {
	t.Run("UpdateBalance - successful positive update", func(t *testing.T) {
		b := domain.NewBalance("user-123", 500)
		originalUpdatedAt := b.UpdatedAt

		// Add a tiny sleep to guarantee UpdatedAt actually changes
		time.Sleep(1 * time.Millisecond)

		updatedB, err := b.UpdateBalance(100)
		assert.NoError(t, err)
		assert.NotNil(t, updatedB)
		assert.Equal(t, int64(600), updatedB.CurrentBalance)
		assert.Equal(t, int64(600), b.CurrentBalance) // mutated in-place
		assert.True(t, b.UpdatedAt.After(originalUpdatedAt))
	})

	t.Run("UpdateBalance - successful negative update", func(t *testing.T) {
		b := domain.NewBalance("user-123", 500)
		originalUpdatedAt := b.UpdatedAt

		time.Sleep(1 * time.Millisecond)

		updatedB, err := b.UpdateBalance(-100)
		assert.NoError(t, err)
		assert.NotNil(t, updatedB)
		assert.Equal(t, int64(400), updatedB.CurrentBalance)
		assert.True(t, b.UpdatedAt.After(originalUpdatedAt))
	})

	t.Run("UpdateBalance - error on zero amount", func(t *testing.T) {
		b := domain.NewBalance("user-123", 500)
		updatedB, err := b.UpdateBalance(0)
		assert.Nil(t, updatedB)
		assert.ErrorIs(t, err, domain.ErrInvalidBalanceUpdateAmt)
	})

	t.Run("UpdateBalance - error on insufficient balance", func(t *testing.T) {
		b := domain.NewBalance("user-123", 50)
		updatedB, err := b.UpdateBalance(-100)
		assert.Nil(t, updatedB)
		assert.ErrorIs(t, err, domain.ErrInsufficientBalance)
		assert.Equal(t, int64(50), b.CurrentBalance) // remains unchanged
	})
}
