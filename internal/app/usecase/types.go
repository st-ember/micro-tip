package usecase

type OrderStatus string

const (
	OrderStatusPending OrderStatus = "pending"
	OrderStatusFailed  OrderStatus = "failed"
	OrderStatusSuccess OrderStatus = "success"
)
