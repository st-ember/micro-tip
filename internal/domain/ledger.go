package domain

import (
	"time"
)

type LedgerEntry struct {
	ID             string
	UserID         string
	IdempotencyKey string
	TransactionID  string
	Amount         int64
	Type           LedgerEntryType
	CreatedAt      time.Time
}

func NewLedgerEntry(
	id, userID, idpKey, txID string,
	amount int64, entryType LedgerEntryType,
) (*LedgerEntry, error) {
	if amount == 0 {
		return nil, ErrInvalidLedgerAmt
	}

	if !entryType.IsValid() {
		return nil, ErrEntryTypeInvalid
	}

	return &LedgerEntry{
		ID:             id,
		UserID:         userID,
		IdempotencyKey: idpKey,
		TransactionID:  txID,
		Amount:         amount,
		Type:           entryType,
		CreatedAt:      time.Now(),
	}, nil
}
