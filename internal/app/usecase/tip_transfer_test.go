package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	nopmetrics "github.com/st-ember/microtip/internal/adpt/driven/metrics/prometheus/nop"
	"github.com/st-ember/microtip/internal/app/port/cache"
	cacheMocks "github.com/st-ember/microtip/internal/app/port/cache/mocks"
	logMocks "github.com/st-ember/microtip/internal/app/port/log/mocks"
	"github.com/st-ember/microtip/internal/app/port/repo"
	repoMocks "github.com/st-ember/microtip/internal/app/port/repo/mocks"
	"github.com/st-ember/microtip/internal/app/usecase"
	"github.com/st-ember/microtip/internal/domain"
)

func TestTipTransferUsecase(t *testing.T) {
	const (
		platformID       = "platform-789"
		senderID         = "sender-123"
		creatorID        = "creator-456"
		splitBasisPoints = 1525 // 15.25% platform fee
		txID             = "tx-123"
		idpKey           = "idp-123"
		amount           = int64(100)
		cShare           = int64(85) // 100 - (100 * 1525 / 10000) = 100 - 15 = 85
		pShare           = int64(15) // 100 * 1525 / 10000 = 15
	)

	// Healthy default command
	cmd, err := domain.NewTipTransferCommand(senderID, creatorID, idpKey, amount)
	assert.NoError(t, err)

	ctx := context.Background()

	t.Run("Success - cache hits for all balances", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		// Create mock balances
		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		// Set up retrieve balance cache hit expectations
		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(platformBal, nil).Once()

		// Unit of work expectations
		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		// Ledger savings expectations
		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.MatchedBy(func(entry *domain.LedgerEntry) bool {
			return entry.UserID == senderID && entry.Amount == amount && entry.Type == domain.LedgerEntryTypeTipTransfer
		})).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.MatchedBy(func(entry *domain.LedgerEntry) bool {
			return entry.UserID == creatorID && entry.Amount == cShare && entry.Type == domain.LedgerEntryTypeTipTransfer
		})).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.MatchedBy(func(entry *domain.LedgerEntry) bool {
			return entry.UserID == platformID && entry.Amount == pShare && entry.Type == domain.LedgerEntryTypeTipTransfer
		})).Return(nil).Once()

		// Balance savings expectations
		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == senderID && b.CurrentBalance == 900
		})).Return(nil).Once()

		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == creatorID && b.CurrentBalance == 585
		})).Return(nil).Once()

		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID && b.CurrentBalance == 115
		})).Return(nil).Once()

		// Commit expectation
		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		// Cache updates expectations
		cacheMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == senderID && b.CurrentBalance == 900
		})).Return(nil).Once()

		cacheMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == creatorID && b.CurrentBalance == 585
		})).Return(nil).Once()

		cacheMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID && b.CurrentBalance == 115
		})).Return(nil).Once()

		// Execute
		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.NoError(t, err)
	})

	t.Run("Success - cache misses (fallback to DB) for all balances", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		// Cache misses
		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(nil, redis.Nil).Once()
		bRepoReadOnlyMock.EXPECT().GetBalance(ctx, senderID).Return(senderBal, nil).Once()

		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(nil, redis.Nil).Once()
		bRepoReadOnlyMock.EXPECT().GetBalance(ctx, creatorID).Return(creatorBal, nil).Once()

		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(nil, nil).Once() // testing nil, nil cache miss too
		bRepoReadOnlyMock.EXPECT().GetBalance(ctx, platformID).Return(platformBal, nil).Once()

		// Unit of work
		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		// Ledger
		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.Anything).Return(nil).Times(3)

		// Balance save
		bRepoMock.EXPECT().SaveBalance(ctx, mock.Anything).Return(nil).Times(3)

		// Commit
		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		// Cache save
		cacheMock.EXPECT().SaveBalance(ctx, mock.Anything).Return(nil).Times(3)

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.NoError(t, err)
	})

	t.Run("Error - retrieve sender balance fails due to Redis error", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		redisErr := errors.New("redis connection refused")
		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(nil, redisErr).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retrieve sender balance: read cache for user sender-123")
	})

	t.Run("Error - retrieve sender balance fails due to DB error", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(nil, redis.Nil).Once()
		dbErr := errors.New("db table locked")
		bRepoReadOnlyMock.EXPECT().GetBalance(ctx, senderID).Return(nil, dbErr).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retrieve sender balance: get balance from db for user sender-123")
	})

	t.Run("Error - sender current balance is zero or negative", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 0)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient balance for sender sender-123: have 0, need 100")
	})

	t.Run("Error - sender insufficient balance for tip amount", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 50) // less than tip amount (100)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient balance for sender sender-123: have 50, need 100")
	})

	t.Run("Error - retrieve creator balance fails due to DB error", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(nil, redis.Nil).Once()
		dbErr := errors.New("creator db error")
		bRepoReadOnlyMock.EXPECT().GetBalance(ctx, creatorID).Return(nil, dbErr).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retrieve creator balance: get balance from db for user creator-456")
	})

	t.Run("Error - retrieve platform balance fails due to DB error", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(nil, redis.Nil).Once()
		dbErr := errors.New("platform db error")
		bRepoReadOnlyMock.EXPECT().GetBalance(ctx, platformID).Return(nil, dbErr).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retrieve platform balance: get balance from db for user platform-789")
	})

	t.Run("Error - save sender ledger entry fails", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(platformBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		dbErr := errors.New("sender ledger save error")
		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.MatchedBy(func(entry *domain.LedgerEntry) bool {
			return entry.UserID == senderID
		})).Return(dbErr).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save sender ledger entry")
	})

	t.Run("Error - save creator ledger entry fails", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(platformBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.MatchedBy(func(entry *domain.LedgerEntry) bool {
			return entry.UserID == senderID
		})).Return(nil).Once()

		dbErr := errors.New("creator ledger save error")
		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.MatchedBy(func(entry *domain.LedgerEntry) bool {
			return entry.UserID == creatorID
		})).Return(dbErr).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save creator ledger entry")
	})

	t.Run("Error - save platform ledger entry fails", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(platformBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.MatchedBy(func(entry *domain.LedgerEntry) bool {
			return entry.UserID == senderID
		})).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.MatchedBy(func(entry *domain.LedgerEntry) bool {
			return entry.UserID == creatorID
		})).Return(nil).Once()

		dbErr := errors.New("platform ledger save error")
		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.MatchedBy(func(entry *domain.LedgerEntry) bool {
			return entry.UserID == platformID
		})).Return(dbErr).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save platform ledger entry")
	})

	t.Run("Error - save sender balance fails", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(platformBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.Anything).Return(nil).Times(3)

		dbErr := errors.New("sender balance save error")
		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == senderID
		})).Return(dbErr).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save sender balance")
	})

	t.Run("Error - save creator balance fails", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(platformBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.Anything).Return(nil).Times(3)

		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == senderID
		})).Return(nil).Once()

		dbErr := errors.New("creator balance save error")
		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == creatorID
		})).Return(dbErr).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save creator balance")
	})

	t.Run("Error - save platform balance fails", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(platformBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.Anything).Return(nil).Times(3)

		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == senderID
		})).Return(nil).Once()

		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == creatorID
		})).Return(nil).Once()

		dbErr := errors.New("platform balance save error")
		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID
		})).Return(dbErr).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save platform balance")
	})

	t.Run("Error - commit transaction fails", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(platformBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.Anything).Return(nil).Times(3)
		bRepoMock.EXPECT().SaveBalance(ctx, mock.Anything).Return(nil).Times(3)

		commitErr := errors.New("serialization failure")
		uowMock.EXPECT().Commit(ctx).Return(commitErr).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "commit transaction: serialization failure")
	})

	t.Run("Error - cache save balance fails (triggers invalidate)", func(t *testing.T) {
		uowMock := repoMocks.NewMockUnitOfWork(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		cacheMock := cacheMocks.NewMockCache(t)
		bRepoReadOnlyMock := repoMocks.NewMockBalanceRepo(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		senderBal := domain.NewBalance(senderID, 1000)
		creatorBal := domain.NewBalance(creatorID, 500)
		platformBal := domain.NewBalance(platformID, 100)

		cacheMock.EXPECT().ReadBalance(ctx, senderID).Return(senderBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, creatorID).Return(creatorBal, nil).Once()
		cacheMock.EXPECT().ReadBalance(ctx, platformID).Return(platformBal, nil).Once()

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		lRepoMock.EXPECT().SaveLedgerEntry(ctx, mock.Anything).Return(nil).Times(3)
		bRepoMock.EXPECT().SaveBalance(ctx, mock.Anything).Return(nil).Times(3)
		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		// Sender cache save fails -> triggers InvalidateKey
		cacheErr := errors.New("redis OOM")
		cacheMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == senderID
		})).Return(cacheErr).Once()

		cacheMock.EXPECT().InvalidateKey(ctx, senderID).Return(nil).Once()

		// Creator and platform cache saves still execute
		cacheMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == creatorID
		})).Return(nil).Once()

		cacheMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID
		})).Return(nil).Once()

		tu := newTipTransferUsecase(t, uowfMock, cacheMock, bRepoReadOnlyMock, splitBasisPoints, platformID)
		err := tu.Execute(ctx, cmd)
		assert.NoError(t, err)
	})
}

func newTipTransferUsecase(t *testing.T, uowf repo.UnitOfWorkFactory, cache cache.Cache, bRepo repo.BalanceRepo, splitBasisPoints int64, platformID string) usecase.TipTransferUsecase {
	nopMet := nopmetrics.NewNopMetrics()
	mockLog := logMocks.NewMockLogger(t)
	mockLog.EXPECT().ErrorCtx(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLog.EXPECT().ErrorCtx(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLog.EXPECT().WarnCtx(mock.Anything, mock.Anything).Maybe()
	mockLog.EXPECT().WarnCtx(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLog.EXPECT().InfoCtx(mock.Anything, mock.Anything).Maybe()
	mockLog.EXPECT().InfoCtx(mock.Anything, mock.Anything, mock.Anything).Maybe()
	return usecase.NewTipTransferUsecase(uowf, cache, bRepo, nopMet, mockLog, splitBasisPoints, platformID)
}
