package repo

import (
	"context"

	"github.com/st-ember/microtip/internal/domain"
)

type BalanceRepo interface {
	GetBalance(ctx context.Context, userID string) (*domain.Balance, error)
	SaveBalance(ctx context.Context, balance *domain.Balance) error
}
