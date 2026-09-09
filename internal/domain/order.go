package domain

import (
	"errors"
	"time"
)

type Order struct {
	UserID         string
	MerchantradeNo string
	Amount         int64
	Status         OrderStatus
	CreatedAt      time.Time
}

func NewOrder(userID, merchantradeNo string, amount int64) *Order {
	return &Order{
		UserID:         userID,
		MerchantradeNo: merchantradeNo,
		Amount:         amount,
		Status:         OrderStatusPending,
		CreatedAt:      time.Now(),
	}
}

func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending
}

func (o *Order) SetSuccess() (*Order, error) {
	if !o.IsPending() {
		return nil, errors.New("")
	}

	o.Status = OrderStatusSuccess
	return o, nil
}

func (o *Order) SetFailed() (*Order, error) {
	if !o.IsPending() {
		return nil, errors.New("")
	}

	o.Status = OrderStatusFailed
	return o, nil
}
