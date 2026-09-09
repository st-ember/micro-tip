package usecase

import (
	"context"
	"fmt"

	"github.com/st-ember/microtip/internal/app/port/cache"
)

type FailTopupUsecase interface {
	Execute(ctx context.Context, merchantTradeNo string) error
}

type failTopupUsecase struct {
	cache cache.Cache
}

func NewFailTopupUsecase(cache cache.Cache) FailTopupUsecase {
	return &failTopupUsecase{cache}
}

func (fu *failTopupUsecase) Execute(ctx context.Context, merchantTradeNo string) error {
	o, err := fu.cache.ReadOrder(ctx, merchantTradeNo)
	if err != nil {
		return fmt.Errorf("read order from cache: %w", err)
	}

	if !o.IsPending() {
		return nil
	}

	updatedO, err := o.SetFailed()
	if err != nil {
		// Dead code currently. Keeping to future proof
		return fmt.Errorf("set order failed: %w", err)
	}

	if err := fu.cache.SaveOrder(ctx, updatedO); err != nil {
		return fmt.Errorf("save order to cache: %w", err)
	}

	return nil
}
