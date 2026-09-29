package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/st-ember/microtip/internal/app/port/repo"
	"github.com/st-ember/microtip/internal/domain"
)

type PostgresBalanceRepo struct {
	q QueryExecutor
}

// QueryExecutor is used to execute queries with db conn as well as transaction
type QueryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func NewPostgresBalanceRepo(db *sql.DB) repo.BalanceRepo {
	return &PostgresBalanceRepo{db}
}

func NewPostgresBalanceRepoWithTransaction(tx *sql.Tx) repo.BalanceRepo {
	return &PostgresBalanceRepo{tx}
}

func (br *PostgresBalanceRepo) GetBalance(ctx context.Context, userID string) (*domain.Balance, error) {
	b := &domain.Balance{}
	query := `SELECT * FROM balance WHERE user_id = $1;`

	err := br.q.QueryRowContext(ctx, query, userID).Scan(
		&b.UserID,
		&b.CurrentBalance,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}

		return nil, fmt.Errorf("scan balance: %w", err)
	}

	return b, nil
}

func (br *PostgresBalanceRepo) SaveBalance(ctx context.Context, balance *domain.Balance) error {
	query := `
		INSERT INTO balance (user_id, current_balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
		current_balance = EXCLUDED.current_balance,
		updated_at = EXCLUDED.updated_at;
	`

	_, err := br.q.ExecContext(ctx, query, balance.UserID, balance.CurrentBalance, balance.CreatedAt, balance.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save balance: %w", err)
	}

	return nil
}

// SeedInitialBalances populates test accounts with an initial balance for testing
func (br *PostgresBalanceRepo) SeedInitialBalances(ctx context.Context, initialAmount int64, prefix string) error {
	now := time.Now()
	for i := range 100 {
		userID := fmt.Sprintf("%s_%d", prefix, i)
		balance := &domain.Balance{
			UserID:         userID,
			CurrentBalance: initialAmount,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if err := br.SaveBalance(ctx, balance); err != nil {
			return fmt.Errorf("failed to seed balance for %s: %w", userID, err)
		}
	}

	for i := range 50 {
		userID := fmt.Sprintf("creator_%d", i)
		balance := &domain.Balance{
			UserID:         userID,
			CurrentBalance: initialAmount,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if err := br.SaveBalance(ctx, balance); err != nil {
			return fmt.Errorf("failed to seed balance for %s: %w", userID, err)
		}
	}

	balance := &domain.Balance{
		UserID:         "platform-789",
		CurrentBalance: initialAmount,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := br.SaveBalance(ctx, balance); err != nil {
		return fmt.Errorf("failed to seed balance for %s: %w", "platform-789", err)
	}

	return nil
}
