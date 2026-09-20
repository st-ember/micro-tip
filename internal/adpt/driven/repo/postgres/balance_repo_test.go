package postgres_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/st-ember/microtip/internal/adpt/driven/repo/postgres"
	"github.com/st-ember/microtip/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBalance(t *testing.T) {
	tx := beginTx(t)
	repo := postgres.NewPostgresBalanceRepoWithTransaction(tx)
	ctx := t.Context()

	t.Run("not found", func(t *testing.T) {
		balance, err := repo.GetBalance(ctx, "non-existent")
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, balance)
	})

	t.Run("success", func(t *testing.T) {
		userID := "user-1"
		createdAt := time.Now().UTC().Round(time.Microsecond)
		updatedAt := time.Now().UTC().Round(time.Microsecond)

		_, err := tx.ExecContext(ctx,
			"INSERT INTO balance (user_id, current_balance, created_at, updated_at) VALUES ($1, $2, $3, $4)",
			userID, int64(1000), createdAt, updatedAt,
		)
		require.NoError(t, err)

		balance, err := repo.GetBalance(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, balance)
		assert.Equal(t, userID, balance.UserID)
		assert.Equal(t, int64(1000), balance.CurrentBalance)
		assert.True(t, createdAt.Equal(balance.CreatedAt))
		assert.True(t, updatedAt.Equal(balance.UpdatedAt))
	})
}

func TestSaveBalance(t *testing.T) {
	tx := beginTx(t)
	repo := postgres.NewPostgresBalanceRepoWithTransaction(tx)
	ctx := t.Context()

	userID := "user-2"
	balance := &domain.Balance{
		UserID:         userID,
		CurrentBalance: 500,
		CreatedAt:      time.Now().UTC().Round(time.Microsecond),
		UpdatedAt:      time.Now().UTC().Round(time.Microsecond),
	}

	t.Run("insert", func(t *testing.T) {
		err := repo.SaveBalance(ctx, balance)
		assert.NoError(t, err)

		var currentBalance int64
		err = tx.QueryRowContext(ctx, "SELECT current_balance FROM balance WHERE user_id = $1", userID).Scan(&currentBalance)
		assert.NoError(t, err)
		assert.Equal(t, int64(500), currentBalance)
	})

	t.Run("update on conflict", func(t *testing.T) {
		// Save again with updated balance
		balance.CurrentBalance = 800
		balance.UpdatedAt = time.Now().UTC().Round(time.Microsecond)

		err := repo.SaveBalance(ctx, balance)
		assert.NoError(t, err)

		var currentBalance int64
		err = tx.QueryRowContext(ctx, "SELECT current_balance FROM balance WHERE user_id = $1", userID).Scan(&currentBalance)
		assert.NoError(t, err)
		assert.Equal(t, int64(800), currentBalance)
	})
}
