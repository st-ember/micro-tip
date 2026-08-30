package domain

type TipTransferCommand struct {
	SenderID       string
	CreatorID      string
	IdempotencyKey string
	Amount         int64
}

func NewTipTransferCommand(senderID, creatorID, idpKey string, amount int64) (*TipTransferCommand, error) {
	if senderID == creatorID {
		return nil, ErrTipSelf
	}

	if amount <= 0 {
		return nil, ErrInvalidTipAmt
	}

	return &TipTransferCommand{
		SenderID:       senderID,
		CreatorID:      creatorID,
		IdempotencyKey: idpKey,
		Amount:         amount,
	}, nil
}
