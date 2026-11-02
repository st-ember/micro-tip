package syncplatformbalance

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	logMocks "github.com/st-ember/microtip/internal/app/port/log/mocks"
	repoMocks "github.com/st-ember/microtip/internal/app/port/repo/mocks"
	"github.com/st-ember/microtip/internal/domain"
)

func TestNewSyncPlatformBalanceWorker(t *testing.T) {
	loggerMock := logMocks.NewMockLogger(t)
	uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
	platformID := "platform-789"

	t.Run("success with positive interval", func(t *testing.T) {
		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, 600, platformID)
		assert.NoError(t, err)
		assert.NotNil(t, worker)
		assert.Equal(t, int64(600), worker.syncIntervalSec)
		assert.Equal(t, platformID, worker.platformID)
	})

	t.Run("error with zero interval", func(t *testing.T) {
		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, 0, platformID)
		assert.Error(t, err)
		assert.Nil(t, worker)
		assert.Equal(t, "sync interval must be a positive value", err.Error())
	})

	t.Run("error with negative interval", func(t *testing.T) {
		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, -10, platformID)
		assert.Error(t, err)
		assert.Nil(t, worker)
		assert.Equal(t, "sync interval must be a positive value", err.Error())
	})
}

func TestSyncPlatformBalanceWorker_Sync(t *testing.T) {
	const (
		platformID      = "platform-789"
		syncIntervalSec = int64(600)
	)

	ctx := context.Background()

	t.Run("success - existing platform balance", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		currentBal := domain.NewBalance(platformID, 1000)
		sumAmount := int64(250)

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(currentBal, nil).Once()

		lRepoMock.EXPECT().SumLedgerByUserIDWithCutOffTime(ctx, platformID, mock.MatchedBy(func(cutoff time.Time) bool {
			expectedCutoff := time.Now().Add(-time.Duration(syncIntervalSec) * time.Second)
			return cutoff.Sub(expectedCutoff).Abs() < 2*time.Second
		})).Return(sumAmount, nil).Once()

		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID && b.CurrentBalance == 1250
		})).Return(nil).Once()

		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.NoError(t, err)
	})

	t.Run("success - platform balance not found initially (sql.ErrNoRows)", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		sumAmount := int64(500)

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(nil, sql.ErrNoRows).Once()
		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID && b.CurrentBalance == 0
		})).Return(nil).Once()

		lRepoMock.EXPECT().SumLedgerByUserIDWithCutOffTime(ctx, platformID, mock.Anything).Return(sumAmount, nil).Once()

		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID && b.CurrentBalance == 500
		})).Return(nil).Once()

		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.NoError(t, err)
	})

	t.Run("error - start unit of work fails", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)

		uowErr := errors.New("cannot begin transaction")
		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(nil, uowErr).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "start unit of work: cannot begin transaction")
	})

	t.Run("error - get platform balance fails with DB error", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		dbErr := errors.New("db query error")
		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(nil, dbErr).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "get platform balance: db query error")
	})

	t.Run("error - create new platform balance fails", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(nil, sql.ErrNoRows).Once()

		dbErr := errors.New("insert balance failed")
		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID && b.CurrentBalance == 0
		})).Return(dbErr).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create new platform balance: insert balance failed")
	})

	t.Run("error - sum platform balance fails", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		currentBal := domain.NewBalance(platformID, 1000)
		dbErr := errors.New("sum query failed")

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(currentBal, nil).Once()
		lRepoMock.EXPECT().SumLedgerByUserIDWithCutOffTime(ctx, platformID, mock.Anything).Return(0, dbErr).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sum platform balance: sum query failed")
	})

	t.Run("error - update platform balance fails (zero sum)", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		currentBal := domain.NewBalance(platformID, 1000)

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(currentBal, nil).Once()
		// Amount is 0 -> domain.UpdateBalance returns ErrInvalidBalanceUpdateAmt
		lRepoMock.EXPECT().SumLedgerByUserIDWithCutOffTime(ctx, platformID, mock.Anything).Return(0, nil).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update platform balance: balance update amount cannot be zero")
	})

	t.Run("error - save updated platform balance fails", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		currentBal := domain.NewBalance(platformID, 1000)
		dbErr := errors.New("save updated balance failed")

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(currentBal, nil).Once()
		lRepoMock.EXPECT().SumLedgerByUserIDWithCutOffTime(ctx, platformID, mock.Anything).Return(150, nil).Once()
		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID && b.CurrentBalance == 1150
		})).Return(dbErr).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save updated platform balance: save updated balance failed")
	})

	t.Run("error - commit transaction fails", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		currentBal := domain.NewBalance(platformID, 1000)
		commitErr := errors.New("commit failed")

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()

		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(currentBal, nil).Once()
		lRepoMock.EXPECT().SumLedgerByUserIDWithCutOffTime(ctx, platformID, mock.Anything).Return(150, nil).Once()
		bRepoMock.EXPECT().SaveBalance(ctx, mock.MatchedBy(func(b *domain.Balance) bool {
			return b.UserID == platformID && b.CurrentBalance == 1150
		})).Return(nil).Once()
		uowMock.EXPECT().Commit(ctx).Return(commitErr).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "commit transaction: commit failed")
	})

	t.Run("rollback error is logged", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		rollbackErr := errors.New("rollback connection error")
		queryErr := errors.New("query error")

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(rollbackErr).Once()

		// logger.ErrorCtx should log the rollback failure
		loggerMock.EXPECT().ErrorCtx(ctx, "roll back unit of work", rollbackErr).Once()

		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(nil, queryErr).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		err = worker.sync(ctx)
		assert.Error(t, err)
	})
}

func TestSyncPlatformBalanceWorker_Start(t *testing.T) {
	const (
		platformID      = "platform-789"
		syncIntervalSec = int64(600)
	)

	t.Run("success initial sync and stops on context done", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		ctx, cancel := context.WithCancel(context.Background())

		currentBal := domain.NewBalance(platformID, 1000)

		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()
		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(currentBal, nil).Once()
		lRepoMock.EXPECT().SumLedgerByUserIDWithCutOffTime(ctx, platformID, mock.Anything).Return(100, nil).Once()
		bRepoMock.EXPECT().SaveBalance(ctx, mock.Anything).Return(nil).Once()
		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		// Cancel context immediately so Start exits after initial sync
		cancel()
		worker.Start(ctx)
	})

	t.Run("logs error on initial sync failure and stops on context done", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)

		ctx, cancel := context.WithCancel(context.Background())

		syncErr := errors.New("db unavailable")
		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(nil, syncErr).Once()

		loggerMock.EXPECT().ErrorCtx(ctx, "sync platform balance", mock.MatchedBy(func(err error) bool {
			return err != nil && errors.Is(err, syncErr)
		})).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, syncIntervalSec, platformID)
		assert.NoError(t, err)

		cancel()
		worker.Start(ctx)
	})

	t.Run("ticker tick executes sync and logs error if it fails", func(t *testing.T) {
		loggerMock := logMocks.NewMockLogger(t)
		uowfMock := repoMocks.NewMockUnitOfWorkFactory(t)
		uowMock := repoMocks.NewMockUnitOfWork(t)
		bRepoMock := repoMocks.NewMockBalanceRepo(t)
		lRepoMock := repoMocks.NewMockLedgerRepo(t)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentBal := domain.NewBalance(platformID, 1000)

		// 1. Initial sync succeeds
		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(uowMock, nil).Once()
		uowMock.EXPECT().LedgerRepo().Return(lRepoMock).Once()
		uowMock.EXPECT().BalanceRepo().Return(bRepoMock).Once()
		uowMock.EXPECT().Rollback(ctx).Return(nil).Once()
		bRepoMock.EXPECT().GetBalance(ctx, platformID).Return(currentBal, nil).Once()
		lRepoMock.EXPECT().SumLedgerByUserIDWithCutOffTime(ctx, platformID, mock.Anything).Return(100, nil).Once()
		bRepoMock.EXPECT().SaveBalance(ctx, mock.Anything).Return(nil).Once()
		uowMock.EXPECT().Commit(ctx).Return(nil).Once()

		// 2. Ticker tick sync fails and logs error
		tickerErr := errors.New("ticker sync failed")
		uowfMock.EXPECT().NewUnitOfWork(ctx).Return(nil, tickerErr).Once()
		loggerMock.EXPECT().ErrorCtx(ctx, "sync platform balance", mock.MatchedBy(func(err error) bool {
			return errors.Is(err, tickerErr)
		})).Once()

		worker, err := NewSyncPlatformBalanceWorker(loggerMock, uowfMock, 1, platformID) // 1 second interval
		assert.NoError(t, err)

		go func() {
			time.Sleep(1100 * time.Millisecond)
			cancel()
		}()

		worker.Start(ctx)
	})
}
