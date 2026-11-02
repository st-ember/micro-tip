package syncplatformbalance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/st-ember/microtip/internal/app/port/log"
	"github.com/st-ember/microtip/internal/app/port/repo"
	"github.com/st-ember/microtip/internal/domain"
)

type SyncPlatformBalanceWorker struct {
	logger          log.Logger
	uowF            repo.UnitOfWorkFactory
	syncIntervalSec int64
	platformID      string
}

func NewSyncPlatformBalanceWorker(
	logger log.Logger, uowFactory repo.UnitOfWorkFactory, syncIntervalSec int64, platformID string,
) (*SyncPlatformBalanceWorker, error) {
	if syncIntervalSec <= 0 {
		return nil, errors.New("sync interval must be a positive value")
	}

	return &SyncPlatformBalanceWorker{logger, uowFactory, syncIntervalSec, platformID}, nil
}

func (sw *SyncPlatformBalanceWorker) Start(ctx context.Context) {
	if err := sw.sync(ctx); err != nil {
		sw.logger.ErrorCtx(ctx, "sync platform balance", err)
	}

	ticker := time.NewTicker(time.Duration(sw.syncIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sw.sync(ctx); err != nil {
				sw.logger.ErrorCtx(ctx, "sync platform balance", err)
			}
		}
	}
}

func (sw *SyncPlatformBalanceWorker) sync(ctx context.Context) error {
	uow, err := sw.uowF.NewUnitOfWork(ctx)
	if err != nil {
		return fmt.Errorf("start unit of work: %w", err)
	}
	defer func() {
		if err := uow.Rollback(ctx); err != nil {
			sw.logger.ErrorCtx(ctx, "roll back unit of work", err)
		}
	}()

	lRepo := uow.LedgerRepo()
	bRepo := uow.BalanceRepo()

	currentB, err := bRepo.GetBalance(ctx, sw.platformID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			currentB = domain.NewBalance(sw.platformID, 0)
			if err := bRepo.SaveBalance(ctx, currentB); err != nil {
				return fmt.Errorf("create new platform balance: %w", err)
			}
		} else {
			return fmt.Errorf("get platform balance: %w", err)
		}
	}

	cutoffTime := time.Now().Add(-time.Duration(sw.syncIntervalSec) * time.Second)

	sum, err := lRepo.SumLedgerByUserIDWithCutOffTime(ctx, sw.platformID, cutoffTime)
	if err != nil {
		return fmt.Errorf("sum platform balance: %w", err)
	}

	updatedB, err := currentB.UpdateBalance(sum)
	if err != nil {
		return fmt.Errorf("update platform balance: %w", err)
	}

	if err := bRepo.SaveBalance(ctx, updatedB); err != nil {
		return fmt.Errorf("save updated platform balance: %w", err)
	}

	if err := uow.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
