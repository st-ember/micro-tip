package cache

import (
	"context"

	"github.com/st-ember/microtip/internal/domain"
)

type Cache interface {
	ReadBalance(ctx context.Context, key string) (*domain.Balance, error)
	SaveBalance(ctx context.Context, balance *domain.Balance) error
	InvalidateKey(ctx context.Context, key string) error
}
