package http

import (
	"github.com/gin-gonic/gin"
	"github.com/st-ember/microtip/internal/adpt/driving/http/tip"
	"github.com/st-ember/microtip/internal/app/usecase"
)

type Router struct {
	Engine *gin.Engine
}

func NewRouter(tipTransferUC usecase.TipTransferUsecase) *Router {
	router := gin.Default()

	th := tip.NewTipHandler(tipTransferUC)
	router.POST("/tip-transfer", th.HandleTipTransfer)

	return &Router{
		Engine: router,
	}
}
