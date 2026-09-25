package http

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/st-ember/microtip/internal/adpt/driving/http/middleware"
	"github.com/st-ember/microtip/internal/adpt/driving/http/payment"
	"github.com/st-ember/microtip/internal/adpt/driving/http/tip"
	"github.com/st-ember/microtip/internal/app/port/hash"
	"github.com/st-ember/microtip/internal/app/port/log"
	"github.com/st-ember/microtip/internal/app/port/metrics"
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
	logger log.Logger,
	metrics metrics.Metrics,
	merchantID string,
	tradeDesc string,
	returnURL string,
	clientBackURL string,
	paymentURL string,
) *Router {
	// Ensure no default middleware is used
	router := gin.New()

	// Add back recovery middleware so the server doesn't crash
	router.Use(gin.Recovery())

	// Attach observability middleware
	router.Use(middleware.Observability(metrics, logger))

	// Tip Handler
	th := tip.NewTipHandler(tipTransferUC, logger)
	router.POST("/tip-transfer", th.HandleTipTransfer)

	// Payment Handler
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
		logger,
	)
	router.POST("/checkout", ph.HandleCheckout)
	router.POST("/payment/confirmation", ph.HandleConfirmation)
	router.GET("/payment/status/:merchant_trade_no", ph.HandleOrderStatus)

	// expose metrics for scraping
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return &Router{
		Engine: router,
	}
}
