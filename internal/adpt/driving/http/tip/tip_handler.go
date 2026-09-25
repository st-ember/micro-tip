package tip

import (
	"github.com/st-ember/microtip/internal/app/port/log"
	"github.com/st-ember/microtip/internal/app/usecase"
)

type TipHandler struct {
	tiptransferUC usecase.TipTransferUsecase
	logger        log.Logger
}

func NewTipHandler(tiptransferUC usecase.TipTransferUsecase, logger log.Logger) *TipHandler {
	return &TipHandler{tiptransferUC, logger}
}
