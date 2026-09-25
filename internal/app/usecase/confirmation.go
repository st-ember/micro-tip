package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/st-ember/microtip/internal/app/port/cache"
	"github.com/st-ember/microtip/internal/app/port/hash"
	"github.com/st-ember/microtip/internal/app/port/log"
	"github.com/st-ember/microtip/internal/app/port/repo"
	"github.com/st-ember/microtip/internal/domain"
)

type ConfirmationUsecase interface {
	Execute(ctx context.Context, input ConfirmationInput) error
}

type confirmationUsecase struct {
	uowf   repo.UnitOfWorkFactory
	cache  cache.Cache
	hasher hash.Hasher
	logger log.Logger
}

func NewConfirmationUsecase(
	uowf repo.UnitOfWorkFactory, cache cache.Cache,
	hasher hash.Hasher, logger log.Logger,
) ConfirmationUsecase {
	return &confirmationUsecase{uowf, cache, hasher, logger}
}

func (tu *confirmationUsecase) Execute(ctx context.Context, input ConfirmationInput) error {
	// 1. Check Hash
	ok, err := tu.hasher.ValidateCheckMacVal(input.CheckMacValueParams, input.CheckMacValue)
	if err != nil {
		return fmt.Errorf("validate check mac val: %w", err)
	}
	if !ok {
		return nil
	}

	// 2. Check cached order
	o, err := tu.cache.ReadOrder(ctx, input.MerchantTradeNo)
	if err != nil {
		return fmt.Errorf("read order from cache: %w", err)
	}

	// Early return for already processed order
	if !o.IsPending() {
		return nil
	}

	// 3. Update cache
	updatedO, err := o.SetSuccess()
	if err != nil {
		// Dead code currently. Keeping for future proof
		return fmt.Errorf("set order success: %w", err)
	}

	if err := tu.cache.SaveOrder(ctx, updatedO); err != nil {
		return fmt.Errorf("save order in cache: %w", err)
	}

	// 4. DB writes
	uow, err := tu.uowf.NewUnitOfWork(ctx)
	if err != nil {
		return fmt.Errorf("start unit of work: %w", err)
	}
	defer func() {
		if err := uow.Rollback(ctx); err != nil {
			tu.logger.ErrorCtx(ctx, "roll back transaction", err, "merchant_trade_no", input.MerchantTradeNo)
		}
	}()

	lRepo := uow.LedgerRepo()
	bRepo := uow.BalanceRepo()

	b, err := bRepo.GetBalance(ctx, o.UserID)
	if err != nil {
		// If no prior balance for user, create new balance
		if errors.Is(err, sql.ErrNoRows) {
			b = domain.NewBalance(o.UserID, input.Amount)
		} else {
			return fmt.Errorf("get user balance: %w", err)
		}
	}

	updatedB, err := b.UpdateBalance(input.Amount)
	if err != nil {
		return fmt.Errorf("update user balance: %w", err)
	}

	transactionID := uuid.NewString()

	l, err := domain.NewLedgerEntry(
		uuid.NewString(),
		o.UserID,
		input.MerchantTradeNo, // Acts as Idempotency key
		transactionID,
		input.Amount,
		domain.LedgerEntryTypeTopUp,
	)
	if err != nil {
		return fmt.Errorf("create ledger entry: %w", err)
	}

	// Save to repo
	if err := lRepo.SaveLedgerEntry(ctx, l); err != nil {
		return fmt.Errorf("save ledger entry: %w", err)
	}
	if err := bRepo.SaveBalance(ctx, updatedB); err != nil {
		return fmt.Errorf("save user balance: %w", err)
	}

	// Commit transaction
	if err := uow.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	// Update cache
	if err := tu.cache.SaveBalance(ctx, updatedB); err != nil {
		// Clear cache on error to ensure correct balance is fetched
		if err := tu.cache.InvalidateKey(ctx, o.UserID); err != nil {
			tu.logger.ErrorCtx(ctx, "invalidate balance", err, "user_id", o.UserID)
		}

		return fmt.Errorf("cache balance: %w", err)
	}

	return nil
}
