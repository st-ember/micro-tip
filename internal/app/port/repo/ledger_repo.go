package repo

import (
	"context"
	"time"

	"github.com/st-ember/microtip/internal/domain"
)

type LedgerRepo interface {
	SaveLedgerEntry(ctx context.Context, entry *domain.LedgerEntry) error
	SumLedgerByUserIDWithCutOffTime(ctx context.Context, userID string, cutoffTime time.Time) (int64, error)
}
