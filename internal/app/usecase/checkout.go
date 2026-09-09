package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/st-ember/microtip/internal/app/port/cache"
	"github.com/st-ember/microtip/internal/app/port/hash"
	"github.com/st-ember/microtip/internal/domain"
)

type CheckoutUsecase interface {
	Execute(ctx context.Context, userID string, totalAmount int64) (*CheckoutResult, error)
}

type checkoutUsecase struct {
	cache  cache.Cache
	hasher hash.Hasher
}

func NewCheckoutUsecase(cache cache.Cache, hasher hash.Hasher) CheckoutUsecase {
	return &checkoutUsecase{cache, hasher}
}

func (cu *checkoutUsecase) Execute(ctx context.Context, userID string, totalAmount int64) (*CheckoutResult, error) {
	cleanUUID := strings.ReplaceAll(uuid.NewString(), "-", "")
	merchantTradeNo := cleanUUID[:20]
	o := domain.NewOrder(userID, merchantTradeNo, totalAmount)

	if err := cu.cache.SaveOrder(ctx, o); err != nil {
		return nil, fmt.Errorf("save order in cache: %w", err)
	}

	return &CheckoutResult{
		MerchantTradeNo:   merchantTradeNo,
		MerchantTradeDate: time.Now().Format("2006/01/02 15:04:05"),
	}, nil
}
