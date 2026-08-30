package domain

import (
	"time"
)

type Balance struct {
	UserID         string
	CurrentBalance int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewBalance(userID string, startingBalance int64) *Balance {
	return &Balance{
		UserID:         userID,
		CurrentBalance: startingBalance,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func (b *Balance) UpdateBalance(amount int64) (*Balance, error) {
	if amount == 0 {
		return nil, ErrInvalidBalanceUpdateAmt
	}

	newBalance := b.CurrentBalance + amount

	if newBalance < 0 {
		return nil, ErrInsufficientBalance
	}

	// Update current balance and update time
	b.CurrentBalance = newBalance
	b.UpdatedAt = time.Now()

	return b, nil
}
