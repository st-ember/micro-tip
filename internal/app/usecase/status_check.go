package usecase

import (
	"context"
	"fmt"

	"github.com/st-ember/microtip/internal/app/port/cache"
	"github.com/st-ember/microtip/internal/domain"
)

type StatusCheckUsecase interface {
	Execute(ctx context.Context, tradeNo string) (domain.OrderStatus, error)
}

type statusCheckUsecase struct {
	cache cache.Cache
}

func NewStatusCheckUsecase(cache cache.Cache) StatusCheckUsecase {
	return &statusCheckUsecase{cache}
}

func (su *statusCheckUsecase) Execute(ctx context.Context, merchantTradeNo string) (domain.OrderStatus, error) {
	order, err := su.cache.ReadOrder(ctx, merchantTradeNo)
	if err != nil {
		return "", fmt.Errorf("retrive order from cache: %w", err)
	}

	return order.Status, nil
}
