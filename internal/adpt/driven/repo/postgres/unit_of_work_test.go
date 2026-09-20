package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/st-ember/microtip/internal/adpt/driven/repo/postgres"
	"github.com/st-ember/microtip/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUnitOfWork(t *testing.T) {
	factory := postgres.NewPostgresUnitOfWorkFactory(TestDB)
	ctx := context.Background()

	uow, err := factory.NewUnitOfWork(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, uow)
	_ = uow.Rollback(ctx)
}

func TestUnitOfWork_Commit(t *testing.T) {
	factory := postgres.NewPostgresUnitOfWorkFactory(TestDB)
	ctx := context.Background()

	uow, err := factory.NewUnitOfWork(ctx)
	require.NoError(t, err)

	// Save some test data
	balance := &domain.Balance{
		UserID:         "uow-commit-user",
		CurrentBalance: 150,
		CreatedAt:      time.Now().UTC().Round(time.Microsecond),
		UpdatedAt:      time.Now().UTC().Round(time.Microsecond),
	}
	err = uow.BalanceRepo().SaveBalance(ctx, balance)
	require.NoError(t, err)

	// Commit UoW
	err = uow.Commit(ctx)
	assert.NoError(t, err)

	// Verify data is committed to global connection
	var currentBalance int64
	err = TestDB.QueryRowContext(ctx, "SELECT current_balance FROM balance WHERE user_id = $1", "uow-commit-user").Scan(&currentBalance)
	assert.NoError(t, err)
	assert.Equal(t, int64(150), currentBalance)

	// Double commit should error
	err = uow.Commit(ctx)
	assert.Error(t, err)
}

func TestUnitOfWork_Rollback(t *testing.T) {
	factory := postgres.NewPostgresUnitOfWorkFactory(TestDB)
	ctx := context.Background()

	uow, err := factory.NewUnitOfWork(ctx)
	require.NoError(t, err)

	// Save some test data
	balance := &domain.Balance{
		UserID:         "uow-rollback-user",
		CurrentBalance: 150,
		CreatedAt:      time.Now().UTC().Round(time.Microsecond),
		UpdatedAt:      time.Now().UTC().Round(time.Microsecond),
	}
	err = uow.BalanceRepo().SaveBalance(ctx, balance)
	require.NoError(t, err)

	// Rollback UoW
	err = uow.Rollback(ctx)
	assert.NoError(t, err)

	// Verify data is NOT committed
	var currentBalance int64
	err = TestDB.QueryRowContext(ctx, "SELECT current_balance FROM balance WHERE user_id = $1", "uow-rollback-user").Scan(&currentBalance)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// Second rollback should ignore and succeed
	err = uow.Rollback(ctx)
	assert.NoError(t, err)
}
