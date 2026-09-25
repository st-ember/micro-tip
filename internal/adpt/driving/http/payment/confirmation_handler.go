package payment

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/st-ember/microtip/internal/app/usecase"
)

func (ph *PaymentHandler) HandleConfirmation(c *gin.Context) {
	var form ECPayReturnForm

	if err := c.ShouldBind(&form); err != nil {
		// Inform ECPay of the anomaly.
		ph.logger.ErrorCtx(c, "invalid json request", err, "handler", "confirmation")
		c.String(http.StatusBadRequest, "0|Invalid Parameters")
		return
	}

	// RtnCode 1 means transaction success.
	if form.RtnCode != 1 {
		// Log failure
		ph.logger.InfoCtx(c, "ecpay notified topup failed", "merchant_tradeno", form.MerchantTradeNo)

		// Execute usecase to invalidate cache
		if err := ph.failTopupUC.Execute(c.Request.Context(), form.MerchantTradeNo); err != nil {
			ph.logger.ErrorCtx(c, "execute usecase", err, "handler", "confirmation", "usecase", "fail_topup")
		}

		// Return confirmation that ECPay did its job and could close this transaction.
		c.String(http.StatusOK, "1|OK")
		return
	}

	input := usecase.ConfirmationInput{
		CheckMacValue:       form.CheckMacValue,
		CheckMacValueParams: form.ToMap(),
		MerchantTradeNo:     form.MerchantTradeNo,
		Amount:              int64(form.TradeAmt),
	}

	// ECPay still did its job, so fall through to confirmation.
	if err := ph.confirmationUsecase.Execute(c.Request.Context(), input); err != nil {
		// Log critical and send alarm for manual audit.
		ph.logger.CriticalCtx(
			c, "execute usecase", err, "handler", "confirmation",
			"usecase", "confirmation", "merchant_trade_no", form.MerchantTradeNo,
		)
	}

	// Return confirmation to ECPay so we don't receive further notices about this transaction.
	c.String(http.StatusOK, "1|OK")
}
