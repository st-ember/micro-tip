package tip

import "github.com/st-ember/microtip/internal/domain"

type TipTransferReq struct {
	SenderID       string `json:"sender_id" binding:"required"`
	CreatorID      string `json:"creator_id" binding:"required"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Amount         int64  `json:"amount" binding:"required"`
}

func (r TipTransferReq) ToCmd() (*domain.TipTransferCommand, error) {
	return domain.NewTipTransferCommand(
		r.SenderID,
		r.CreatorID,
		r.IdempotencyKey,
		r.Amount,
	)
}
