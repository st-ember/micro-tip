package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cacheMocks "github.com/st-ember/microtip/internal/app/port/cache/mocks"
	hashMocks "github.com/st-ember/microtip/internal/app/port/hash/mocks"
	"github.com/st-ember/microtip/internal/app/usecase"
	"github.com/st-ember/microtip/internal/domain"
)

func TestNewCheckoutUsecase(t *testing.T) {
	t.Run("NewCheckoutUsecase - initializes correctly", func(t *testing.T) {
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)

		uc := usecase.NewCheckoutUsecase(cacheMock, hasherMock)
		assert.NotNil(t, uc)
	})
}

func TestCheckoutUsecase_Execute(t *testing.T) {
	const (
		userID      = "user-123"
		totalAmount = int64(1500)
	)

	ctx := context.Background()

	t.Run("Success - creates and saves order with correct ECPay format", func(t *testing.T) {
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)

		// Set expectation on SaveOrder
		cacheMock.EXPECT().
			SaveOrder(ctx, mock.MatchedBy(func(o *domain.Order) bool {
				return o.UserID == userID &&
					o.Amount == totalAmount &&
					o.IsPending() &&
					len(o.MerchantradeNo) == 20 // Compliant 20-character trade number
			})).
			Return(nil).
			Once()

		uc := usecase.NewCheckoutUsecase(cacheMock, hasherMock)

		beforeExecution := time.Now()
		res, err := uc.Execute(ctx, userID, totalAmount)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.MerchantTradeNo, 20)

		// Verify date matches ECPay layout: yyyy/MM/dd HH:mm:ss
		parsedDate, parseErr := time.Parse("2006/01/02 15:04:05", res.MerchantTradeDate)
		assert.NoError(t, parseErr)
		assert.True(t, parsedDate.After(beforeExecution.Add(-1*time.Second)) || parsedDate.Equal(beforeExecution))
	})

	t.Run("Error - failed to save order in cache", func(t *testing.T) {
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)

		cacheErr := errors.New("redis save error")
		cacheMock.EXPECT().
			SaveOrder(ctx, mock.Anything).
			Return(cacheErr).
			Once()

		uc := usecase.NewCheckoutUsecase(cacheMock, hasherMock)
		res, err := uc.Execute(ctx, userID, totalAmount)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "save order in cache:")
	})
}
