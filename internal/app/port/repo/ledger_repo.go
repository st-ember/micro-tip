package repo

import (
	"context"

	"github.com/st-ember/microtip/internal/domain"
)

type LedgerRepo interface {
	SaveLedgerEntry(ctx context.Context, entry *domain.LedgerEntry) error
}
