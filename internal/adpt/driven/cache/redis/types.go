package redis

import (
	"time"

	"github.com/st-ember/microtip/internal/domain"
)

type BalanceCache struct {
	UserID         string    `json:"user_id"`
	CurrentBalance int64     `json:"current_balance"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (bc BalanceCache) ToDomain() *domain.Balance {
	return &domain.Balance{
		UserID:         bc.UserID,
		CurrentBalance: bc.CurrentBalance,
		CreatedAt:      bc.CreatedAt,
		UpdatedAt:      bc.UpdatedAt,
	}
}

type OrderCache struct {
	UserID         string             `json:"user_id"`
	MerchantradeNo string             `json:"merchant_trade_no"`
	Amount         int64              `json:"amount"`
	Status         domain.OrderStatus `json:"status"`
	CreatedAt      time.Time          `json:"created_at"`
}

func (oc OrderCache) ToDomain() *domain.Order {
	return &domain.Order{
		UserID:         oc.UserID,
		MerchantradeNo: oc.MerchantradeNo,
		Amount:         oc.Amount,
		Status:         oc.Status,
		CreatedAt:      oc.CreatedAt,
	}
}
