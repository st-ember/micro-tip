package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	cacheMocks "github.com/st-ember/microtip/internal/app/port/cache/mocks"
	"github.com/st-ember/microtip/internal/app/usecase"
	"github.com/st-ember/microtip/internal/domain"
)

func TestStatusCheckUsecase_Execute(t *testing.T) {
	const (
		userID          = "user-123"
		merchantTradeNo = "ecpay20230312153023"
		amount          = int64(1000)
	)

	ctx := context.Background()

	t.Run("Success - retrieves order status from cache correctly", func(t *testing.T) {
		cacheMock := cacheMocks.NewMockCache(t)

		pendingOrder := domain.NewOrder(userID, merchantTradeNo, amount)
		cacheMock.EXPECT().
			ReadOrder(ctx, merchantTradeNo).
			Return(pendingOrder, nil).
			Once()

		uc := usecase.NewStatusCheckUsecase(cacheMock)
		status, err := uc.Execute(ctx, merchantTradeNo)

		assert.NoError(t, err)
		assert.Equal(t, domain.OrderStatusPending, status)
	})

	t.Run("Error - failed to retrieve order from cache", func(t *testing.T) {
		cacheMock := cacheMocks.NewMockCache(t)

		cacheMock.EXPECT().
			ReadOrder(ctx, merchantTradeNo).
			Return(nil, errors.New("redis cache miss or down")).
			Once()

		uc := usecase.NewStatusCheckUsecase(cacheMock)
		status, err := uc.Execute(ctx, merchantTradeNo)

		assert.Error(t, err)
		assert.Equal(t, domain.OrderStatus(""), status)
		assert.Contains(t, err.Error(), "retrive order from cache:")
	})
}
