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
		c.String(http.StatusBadRequest, "0|Invalid Parameters")
		return
	}

	// RtnCode 1 means transaction success.
	if form.RtnCode != 1 {
		// Execute usecase to invalidate cache
		// TODO: log error
		_ = ph.failTopupUC.Execute(c, form.MerchantTradeNo)

		// TODO: log failed transaction
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

	// TODO: Log critical and send alarm for manual audit.
	// ECPay still did its job, so fall through to confirmation.
	_ = ph.confirmationUsecase.Execute(c, input)

	// Return confirmation to ECPay so we don't receive further notices about this transaction.
	c.String(http.StatusOK, "1|OK")
}
