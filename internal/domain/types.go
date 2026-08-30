package domain

type LedgerEntryType string

const (
	LedgerEntryTypeTopUp       = "TOP_UP"
	LedgerEntryTypeTipTransfer = "TIP_TRANSFER"
)

func (l LedgerEntryType) IsValid() bool {
	return l == LedgerEntryTypeTipTransfer || l == LedgerEntryTypeTopUp
}
