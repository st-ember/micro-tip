package http

import (
	"github.com/gin-gonic/gin"
	"github.com/st-ember/microtip/internal/adpt/driving/http/payment"
	"github.com/st-ember/microtip/internal/adpt/driving/http/tip"
	"github.com/st-ember/microtip/internal/app/port/hash"
	"github.com/st-ember/microtip/internal/app/usecase"
)

type Router struct {
	Engine *gin.Engine
}

func NewRouter(
	tipTransferUC usecase.TipTransferUsecase,
	checkoutUC usecase.CheckoutUsecase,
	failTopupUC usecase.FailTopupUsecase,
	confirmationUC usecase.ConfirmationUsecase,
	statusCheckUC usecase.StatusCheckUsecase,
	hasher hash.Hasher,
	merchantID string,
	tradeDesc string,
	returnURL string,
	clientBackURL string,
	paymentURL string,
) *Router {
	router := gin.Default()

	th := tip.NewTipHandler(tipTransferUC)
	router.POST("/tip-transfer", th.HandleTipTransfer)

	ph := payment.NewPaymentHandler(
		merchantID,
		tradeDesc,
		returnURL,
		clientBackURL,
		paymentURL,
		checkoutUC,
		failTopupUC,
		confirmationUC,
		statusCheckUC,
		hasher,
	)
	router.POST("/checkout", ph.HandleCheckout)
	router.POST("/payment/confirmation", ph.HandleConfirmation)
	router.GET("/payment/status/:merchant_trade_no", ph.HandleOrderStatus)

	return &Router{
		Engine: router,
	}
}
