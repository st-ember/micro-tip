package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/st-ember/microtip/internal/domain"
)

func TestNewTipTransferCommand(t *testing.T) {
	const (
		txID      = "tx-123"
		senderID  = "sender-123"
		creatorID = "creator-456"
		idpKey    = "idp-123"
	)

	t.Run("NewTipTransferCommand - successful creation", func(t *testing.T) {
		cmd, err := domain.NewTipTransferCommand(senderID, creatorID, idpKey, 100)
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, senderID, cmd.SenderID)
		assert.Equal(t, creatorID, cmd.CreatorID)
		assert.Equal(t, idpKey, cmd.IdempotencyKey)
		assert.Equal(t, int64(100), cmd.Amount)
	})

	t.Run("NewTipTransferCommand - error on tipping self", func(t *testing.T) {
		cmd, err := domain.NewTipTransferCommand(senderID, senderID, idpKey, 100)
		assert.Nil(t, cmd)
		assert.ErrorIs(t, err, domain.ErrTipSelf)
	})

	t.Run("NewTipTransferCommand - error on non-positive amount", func(t *testing.T) {
		cmd, err := domain.NewTipTransferCommand(senderID, creatorID, idpKey, 0)
		assert.Nil(t, cmd)
		assert.ErrorIs(t, err, domain.ErrInvalidTipAmt)

		cmd, err = domain.NewTipTransferCommand(senderID, creatorID, idpKey, -50)
		assert.Nil(t, cmd)
		assert.ErrorIs(t, err, domain.ErrInvalidTipAmt)
	})
}
