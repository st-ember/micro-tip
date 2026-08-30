package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/st-ember/microtip/internal/domain"
)

func TestLedgerEntryType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		entryType domain.LedgerEntryType
		expected  bool
	}{
		{"TopUp is valid", domain.LedgerEntryTypeTopUp, true},
		{"TipTransfer is valid", domain.LedgerEntryTypeTipTransfer, true},
		{"Empty type is invalid", domain.LedgerEntryType(""), false},
		{"Random type is invalid", domain.LedgerEntryType("UNKNOWN"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entryType.IsValid())
		})
	}
}
