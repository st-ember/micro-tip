package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/st-ember/microtip/internal/app/port/cache"
	"github.com/st-ember/microtip/internal/app/port/log"
	"github.com/st-ember/microtip/internal/app/port/metrics"
	"github.com/st-ember/microtip/internal/app/port/repo"
	"github.com/st-ember/microtip/internal/domain"
)

type TipTransferUsecase interface {
	Execute(ctx context.Context, cmd *domain.TipTransferCommand) error
}

type tipTransferUsecase struct {
	uowf  repo.UnitOfWorkFactory
	cache cache.Cache
	// Read only repo for retrieveBalance helper
	bRepo   repo.BalanceRepo
	metrics metrics.Metrics
	logger  log.Logger
	// Represents the platform fee percentage
	// Goes down to two points below decimal
	// Example: 1525 = 15.25%
	splitBasisPoints int64
	platformID       string
}

func NewTipTransferUsecase(
	uowf repo.UnitOfWorkFactory, cache cache.Cache, bRepo repo.BalanceRepo,
	metrics metrics.Metrics, logger log.Logger, splitBasisPoints int64, platformID string,
) TipTransferUsecase {
	return &tipTransferUsecase{uowf, cache, bRepo, metrics, logger, splitBasisPoints, platformID}
}

func (tu *tipTransferUsecase) Execute(ctx context.Context, cmd *domain.TipTransferCommand) error {
	// Record tip transfer event start
	tu.metrics.IncTip(metrics.TipStatusStarted)

	// Init db duration start time
	start := time.Now()
	uow, err := tu.uowf.NewUnitOfWork(ctx)
	if err != nil {
		return fmt.Errorf("start unit of work: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			tu.metrics.IncTip(metrics.TipStatusDBErr)
		}
		if err := uow.Rollback(ctx); err != nil {
			tu.logger.ErrorCtx(ctx, "roll back transaction", err, "idempotency_key", cmd.IdempotencyKey)
		}
	}()

	lRepo := uow.LedgerRepo()
	bRepo := uow.BalanceRepo()

	// Retrieve sender balance from cache
	sb, err := tu.retrieveBalance(ctx, cmd.SenderID)
	if err != nil {
		return fmt.Errorf("retrieve sender balance: %w", err)
	}

	if sb.CurrentBalance < cmd.Amount {
		tu.metrics.IncTip(metrics.TipStatusBalanceErr)
		return fmt.Errorf("insufficient balance for sender %s: have %d, need %d", cmd.SenderID, sb.CurrentBalance, cmd.Amount)
	}

	updatedSB, err := sb.UpdateBalance(-cmd.Amount)
	if err != nil {
		return fmt.Errorf("update sender balance: %w", err)
	}

	cShare, pShare := tu.splitTip(cmd.Amount)
	cb, err := tu.retrieveBalance(ctx, cmd.CreatorID)
	if err != nil {
		return fmt.Errorf("retrieve creator balance: %w", err)
	}
	// Add only, always resulting in no error
	updatedCB, err := cb.UpdateBalance(cShare)
	if err != nil {
		return fmt.Errorf("update creator balance: %w", err)
	}

	platformB, err := tu.retrieveBalance(ctx, tu.platformID)
	if err != nil {
		return fmt.Errorf("retrieve platform balance: %w", err)
	}

	updatedPB, err := platformB.UpdateBalance(pShare)
	if err != nil {
		return fmt.Errorf("update platform balance: %w", err)
	}

	// Generate transactionID
	txID := uuid.NewString()

	// Create ledger entries
	sl, err := domain.NewLedgerEntry(
		uuid.NewString(),
		cmd.SenderID,
		cmd.IdempotencyKey,
		txID,
		cmd.Amount,
		domain.LedgerEntryTypeTipTransfer,
	)
	if err != nil {
		return fmt.Errorf("create sender ledger entry: %w", err)
	}

	cl, err := domain.NewLedgerEntry(
		uuid.NewString(),
		cmd.CreatorID,
		cmd.IdempotencyKey,
		txID,
		cShare,
		domain.LedgerEntryTypeTipTransfer,
	)
	if err != nil {
		return fmt.Errorf("create creator ledger entry: %w", err)
	}

	pl, err := domain.NewLedgerEntry(
		uuid.NewString(),
		tu.platformID,
		cmd.IdempotencyKey,
		txID,
		pShare,
		domain.LedgerEntryTypeTipTransfer,
	)
	if err != nil {
		return fmt.Errorf("create platform ledger entry: %w", err)
	}

	// Save ledger entries
	if err := lRepo.SaveLedgerEntry(ctx, sl); err != nil {
		return fmt.Errorf("save sender ledger entry: %w", err)
	}

	if err := lRepo.SaveLedgerEntry(ctx, cl); err != nil {
		return fmt.Errorf("save creator ledger entry: %w", err)
	}

	if err := lRepo.SaveLedgerEntry(ctx, pl); err != nil {
		return fmt.Errorf("save platform ledger entry: %w", err)
	}

	// Save updated balances
	if err := bRepo.SaveBalance(ctx, updatedSB); err != nil {
		return fmt.Errorf("save sender balance: %w", err)
	}

	if err := bRepo.SaveBalance(ctx, updatedCB); err != nil {
		return fmt.Errorf("save creator balance: %w", err)
	}

	if err := bRepo.SaveBalance(ctx, updatedPB); err != nil {
		return fmt.Errorf("save platform balance: %w", err)
	}

	// Commit transaction
	if err := uow.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true

	// Record db duration
	duration := time.Since(start)
	tu.metrics.ObserveDBTransactionDuration("tip_transfer", duration)
	tu.metrics.IncTip(metrics.TipStatusSuccess)

	// Update cached balances
	if err := tu.saveOrInvalidateCache(ctx, updatedSB); err != nil {
		tu.logger.ErrorCtx(
			ctx, "cache sender balance", err,
			"user_id", cmd.SenderID, "idempotency_key", cmd.IdempotencyKey,
		)
	}

	if err := tu.saveOrInvalidateCache(ctx, updatedCB); err != nil {
		tu.logger.ErrorCtx(
			ctx, "cache creator balance", err,
			"user_id", cmd.CreatorID, "idempotency_key", cmd.IdempotencyKey,
		)
	}

	if err := tu.saveOrInvalidateCache(ctx, updatedPB); err != nil {
		tu.logger.ErrorCtx(
			ctx, "cache platform balance", err,
			"user_id", tu.platformID, "idempotency_key", cmd.IdempotencyKey,
		)
	}

	return nil
}

// retrieveBalance helps encapsulate balance retrieval
// 1. Try to retrieve from cache
// 2. If not in cache, retrieve from DB
func (tu *tipTransferUsecase) retrieveBalance(ctx context.Context, userID string) (*domain.Balance, error) {
	b, err := tu.cache.ReadBalance(ctx, userID)
	if err == nil && b != nil {
		return b, nil
	}

	// If it's a real Redis error (not just a cache miss), handle/return it
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("read cache for user %s: %w", userID, err)
	}

	// Cache miss or nil value -> Fall back to DB
	b, err = tu.bRepo.GetBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get balance from db for user %s: %w", userID, err)
	}

	return b, nil
}

// splitTip calculates and returns the creator's and platform's respective share of the tip
func (tu *tipTransferUsecase) splitTip(amount int64) (cShare, pShare int64) {
	pShare = (amount * tu.splitBasisPoints) / 10000
	cShare = amount - pShare

	return cShare, pShare
}

func (tu *tipTransferUsecase) saveOrInvalidateCache(ctx context.Context, b *domain.Balance) error {
	if err := tu.cache.SaveBalance(ctx, b); err != nil {
		if err := tu.cache.InvalidateKey(ctx, b.UserID); err != nil {
			tu.logger.WarnCtx(ctx, "invalidate balance", err, "user_id", b.UserID)
		}

		return fmt.Errorf("save cache for user %s: %w", b.UserID, err)
	}

	return nil
}
