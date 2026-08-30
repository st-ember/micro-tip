package domain

import "errors"

var (
	ErrTipSelf                 = errors.New("creator cannot tip self")
	ErrInvalidTipAmt           = errors.New("tip amount must be positive")
	ErrInvalidTopupAmt         = errors.New("topup amount must be positive")
	ErrInvalidBalanceUpdateAmt = errors.New("balance update amount cannot be zero")
	ErrInsufficientBalance     = errors.New("insufficient balance")
	ErrInvalidLedgerAmt        = errors.New("ledger entry amount cannot be zero")
	ErrEntryTypeInvalid        = errors.New("ledger entry type is invalid")
)
