package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cacheMocks "github.com/st-ember/microtip/internal/app/port/cache/mocks"
	"github.com/st-ember/microtip/internal/app/usecase"
	"github.com/st-ember/microtip/internal/domain"
)

func TestFailTopupUsecase_Execute(t *testing.T) {
	const (
		userID          = "user-123"
		merchantTradeNo = "ecpay20230312153023"
		amount          = int64(1000)
	)

	ctx := context.Background()

	t.Run("Success - pending order gets set to failed", func(t *testing.T) {
		cacheMock := cacheMocks.NewMockCache(t)

		pendingOrder := domain.NewOrder(userID, merchantTradeNo, amount)
		cacheMock.EXPECT().
			ReadOrder(ctx, merchantTradeNo).
			Return(pendingOrder, nil).
			Once()

		cacheMock.EXPECT().
			SaveOrder(ctx, mock.MatchedBy(func(o *domain.Order) bool {
				return o.UserID == userID && o.MerchantradeNo == merchantTradeNo && o.Status == domain.OrderStatusFailed
			})).
			Return(nil).
			Once()

		uc := usecase.NewFailTopupUsecase(cacheMock)
		err := uc.Execute(ctx, merchantTradeNo)
		assert.NoError(t, err)
	})

	t.Run("Success - already processed order (non-pending) early returns nil", func(t *testing.T) {
		cacheMock := cacheMocks.NewMockCache(t)

		// Create already processed order
		successOrder := domain.NewOrder(userID, merchantTradeNo, amount)
		_, _ = successOrder.SetSuccess()

		cacheMock.EXPECT().
			ReadOrder(ctx, merchantTradeNo).
			Return(successOrder, nil).
			Once()

		// No SaveOrder should be called
		uc := usecase.NewFailTopupUsecase(cacheMock)
		err := uc.Execute(ctx, merchantTradeNo)
		assert.NoError(t, err)
	})

	t.Run("Error - read order from cache fails", func(t *testing.T) {
		cacheMock := cacheMocks.NewMockCache(t)

		cacheMock.EXPECT().
			ReadOrder(ctx, merchantTradeNo).
			Return(nil, errors.New("redis connection timeout")).
			Once()

		uc := usecase.NewFailTopupUsecase(cacheMock)
		err := uc.Execute(ctx, merchantTradeNo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "read order from cache:")
	})

	t.Run("Error - save order to cache fails", func(t *testing.T) {
		cacheMock := cacheMocks.NewMockCache(t)

		pendingOrder := domain.NewOrder(userID, merchantTradeNo, amount)
		cacheMock.EXPECT().
			ReadOrder(ctx, merchantTradeNo).
			Return(pendingOrder, nil).
			Once()

		cacheMock.EXPECT().
			SaveOrder(ctx, mock.Anything).
			Return(errors.New("redis memory limit exceeded")).
			Once()

		uc := usecase.NewFailTopupUsecase(cacheMock)
		err := uc.Execute(ctx, merchantTradeNo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save order to cache:")
	})
}
