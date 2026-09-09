package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cacheMocks "github.com/st-ember/microtip/internal/app/port/cache/mocks"
	hashMocks "github.com/st-ember/microtip/internal/app/port/hash/mocks"
	repoMocks "github.com/st-ember/microtip/internal/app/port/repo/mocks"
	"github.com/st-ember/microtip/internal/app/usecase"
	"github.com/st-ember/microtip/internal/domain"
)

func TestConfirmationUsecase_Execute(t *testing.T) {
	const (
		userID          = "user-123"
		merchantTradeNo = "ecpay20230312153023"
		amount          = int64(1000)
		checkMacValue   = "TEST_CHECK_MAC_VALUE"
	)

	ctx := context.Background()

	input := usecase.ConfirmationInput{
		CheckMacValue:       checkMacValue,
		CheckMacValueParams: map[string]string{"foo": "bar"},
		MerchantTradeNo:     merchantTradeNo,
		Amount:              amount,
	}

	t.Run("Success - valid signature, pending order, existing user balance", func(t *testing.T) {
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		// 1. Signature validation matches
		hasherMock.EXPECT().
			ValidateCheckMacVal(input.CheckMacValueParams, input.CheckMacValue).
			Return(true, nil).
			Once()

		// 2. Read pending order from cache
		pendingOrder := domain.NewOrder(userID, merchantTradeNo, amount)
		cacheMock.EXPECT().
			ReadOrder(ctx, merchantTradeNo).
			Return(pendingOrder, nil).
			Once()

		// 3. Save success order to cache
		cacheMock.EXPECT().
			SaveOrder(ctx, mock.MatchedBy(func(o *domain.Order) bool {
				return o.UserID == userID && o.MerchantradeNo == merchantTradeNo && o.Status == domain.OrderStatusSuccess
			})).
			Return(nil).
			Once()

		// 4. Unit of Work setup
		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		// 5. Balance Retrieval and Save
		existingBalance := domain.NewBalance(userID, 500)
		bRepoMock.EXPECT().
			GetBalance(ctx, userID).
			Return(existingBalance, nil).
			Once()

		bRepoMock.EXPECT().
			SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
				return b.UserID == userID && b.CurrentBalance == 1500 // 500 + 1000
			})).
			Return(nil).
			Once()

		// 6. Ledger entry save
		lRepoMock.EXPECT().
			SaveLedgerEntry(ctx, mock.MatchedBy(func(l *domain.LedgerEntry) bool {
				return l.UserID == userID && l.Amount == amount && l.Type == domain.LedgerEntryTypeTopUp && l.IdempotencyKey == merchantTradeNo
			})).
			Return(nil).
			Once()

		// 7. Commit
		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		// 8. Cache updated balance
		cacheMock.EXPECT().
			SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
				return b.UserID == userID && b.CurrentBalance == 1500
			})).
			Return(nil).
			Once()

		uc := usecase.NewConfirmationUsecase(uowfMock, cacheMock, hasherMock)
		err := uc.Execute(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("Success - valid signature, pending order, first-time user (creates new balance)", func(t *testing.T) {
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		hasherMock.EXPECT().
			ValidateCheckMacVal(input.CheckMacValueParams, input.CheckMacValue).
			Return(true, nil).
			Once()

		pendingOrder := domain.NewOrder(userID, merchantTradeNo, amount)
		cacheMock.EXPECT().ReadOrder(ctx, merchantTradeNo).Return(pendingOrder, nil).Once()
		cacheMock.EXPECT().SaveOrder(ctx, mock.Anything).Return(nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		// GetBalance returns sql.ErrNoRows to signify first-time user
		bRepoMock.EXPECT().
			GetBalance(ctx, userID).
			Return(nil, sql.ErrNoRows).
			Once()

		bRepoMock.EXPECT().
			SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
				return b.UserID == userID && b.CurrentBalance == 2000 // 1000 initial + 1000 update
			})).
			Return(nil).
			Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.Anything).Return(nil).Once()
		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		cacheMock.EXPECT().
			SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
				return b.UserID == userID && b.CurrentBalance == 2000
			})).
			Return(nil).
			Once()

		uc := usecase.NewConfirmationUsecase(uowfMock, cacheMock, hasherMock)
		err := uc.Execute(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("Early Return - invalid hash signature", func(t *testing.T) {
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)

		// Signature validation fails (returns false, nil)
		hasherMock.EXPECT().
			ValidateCheckMacVal(input.CheckMacValueParams, input.CheckMacValue).
			Return(false, nil).
			Once()

		uc := usecase.NewConfirmationUsecase(uowfMock, cacheMock, hasherMock)
		err := uc.Execute(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("Early Return - already processed order (non-pending status)", func(t *testing.T) {
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)

		hasherMock.EXPECT().
			ValidateCheckMacVal(input.CheckMacValueParams, input.CheckMacValue).
			Return(true, nil).
			Once()

		// Cache returns a success (processed) order
		successOrder := domain.NewOrder(userID, merchantTradeNo, amount)
		_, _ = successOrder.SetSuccess()

		cacheMock.EXPECT().
			ReadOrder(ctx, merchantTradeNo).
			Return(successOrder, nil).
			Once()

		uc := usecase.NewConfirmationUsecase(uowfMock, cacheMock, hasherMock)
		err := uc.Execute(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("Error - hash validation fails with error", func(t *testing.T) {
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)

		hasherMock.EXPECT().
			ValidateCheckMacVal(input.CheckMacValueParams, input.CheckMacValue).
			Return(false, errors.New("hashing service offline")).
			Once()

		uc := usecase.NewConfirmationUsecase(uowfMock, cacheMock, hasherMock)
		err := uc.Execute(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validate check mac val:")
	})

	t.Run("Error - read order from cache fails", func(t *testing.T) {
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)

		hasherMock.EXPECT().
			ValidateCheckMacVal(input.CheckMacValueParams, input.CheckMacValue).
			Return(true, nil).
			Once()

		cacheMock.EXPECT().
			ReadOrder(ctx, merchantTradeNo).
			Return(nil, errors.New("redis connection refused")).
			Once()

		uc := usecase.NewConfirmationUsecase(uowfMock, cacheMock, hasherMock)
		err := uc.Execute(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "read order from cache:")
	})

	t.Run("Error - save updated balance to cache fails (triggers invalidate)", func(t *testing.T) {
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		cacheMock := cacheMocks.NewMockCache(t)
		hasherMock := hashMocks.NewMockHasher(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		hasherMock.EXPECT().ValidateCheckMacVal(input.CheckMacValueParams, input.CheckMacValue).Return(true, nil).Once()
		pendingOrder := domain.NewOrder(userID, merchantTradeNo, amount)
		cacheMock.EXPECT().ReadOrder(ctx, merchantTradeNo).Return(pendingOrder, nil).Once()
		cacheMock.EXPECT().SaveOrder(ctx, mock.Anything).Return(nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		bRepoMock.EXPECT().GetBalance(ctx, userID).Return(nil, sql.ErrNoRows).Once()
		bRepoMock.EXPECT().SaveBalance(ctx, mock.Anything).Return(nil).Once()
		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.Anything).Return(nil).Once()
		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		// Cache saving of updated balance fails
		cacheMock.EXPECT().
			SaveBalance(ctx, mock.Anything).
			Return(errors.New("cache write timeout")).
			Once()

		// Should invalidate cache key for safety
		cacheMock.EXPECT().
			InvalidateKey(ctx, userID).
			Return(nil).
			Once()

		uc := usecase.NewConfirmationUsecase(uowfMock, cacheMock, hasherMock)
		err := uc.Execute(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache balance:")
	})
}
