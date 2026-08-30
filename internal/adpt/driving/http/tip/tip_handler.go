package tip

import (
	"github.com/st-ember/microtip/internal/app/usecase"
)

type TipHandler struct {
	tiptransferUC usecase.TipTransferUsecase
}

func NewTipHandler(tiptransferUC usecase.TipTransferUsecase) *TipHandler {
	return &TipHandler{tiptransferUC}
}
