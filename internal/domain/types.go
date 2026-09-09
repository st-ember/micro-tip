package domain

type LedgerEntryType string

const (
	LedgerEntryTypeTopUp       = "TOP_UP"
	LedgerEntryTypeTipTransfer = "TIP_TRANSFER"
)

func (l LedgerEntryType) IsValid() bool {
	return l == LedgerEntryTypeTipTransfer || l == LedgerEntryTypeTopUp
}

type OrderStatus string

const (
	OrderStatusFailed  OrderStatus = "failed"
	OrderStatusPending OrderStatus = "pending"
	OrderStatusSuccess OrderStatus = "success"
)

func (o OrderStatus) IsValid() bool {
	switch o {
	case OrderStatusFailed:
		return true
	case OrderStatusPending:
		return true
	case OrderStatusSuccess:
		return true
	default:
		return false
	}
}
