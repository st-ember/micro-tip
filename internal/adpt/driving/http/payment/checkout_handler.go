package payment

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (ph *PaymentHandler) HandleCheckout(c *gin.Context) {
	var req CheckoutReq

	// Decipher request
	if err := c.BindJSON(&req); err != nil {
		ph.logger.ErrorCtx(c, "invalid json request", err, "handler", "checkout")
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid input format")
		return
	}

	// Execute usecase
	res, err := ph.checkoutUC.Execute(c.Request.Context(), req.UserID, req.TotalAmount)
	if err != nil {
		ph.logger.ErrorCtx(c, "execute usecase", err, "handler", "checkout", "usecase", "checkout")
		c.AbortWithStatusJSON(http.StatusInternalServerError, "internal server error")
		return
	}

	// Build response
	p := ECPayPaymentParams{
		MerchantID:        ph.merchantID,
		MerchantTradeNo:   res.MerchantTradeNo,
		MerchantTradeDate: res.MerchantTradeDate,
		PaymentType:       PaymentTypeAIO,
		TotalAmount:       req.TotalAmount,
		TradeDesc:         ph.tradeDesc,
		ItemName:          req.ItemName,
		ReturnURL:         ph.returnURL,
		ChoosePayment:     ChoosePaymentCredit,
		EncryptType:       EncryptTypeSHA256,
		ClientBackURL:     ph.clientBackURL,
	}

	// Generate CheckMacValue
	mac, err := ph.hasher.GenerateCheckMacVal(p.ToMap())
	if err != nil {
		ph.logger.ErrorCtx(c, "generate make check mac value", err, "handler", "checkout")
		c.AbortWithStatusJSON(http.StatusInternalServerError, "failed to generate payment signature")
		return
	}
	p.CheckMacValue = mac

	resp := ECPayCheckoutResponse{
		PaymentURL: ph.paymentURL,
		Params:     p,
	}

	// Return response
	c.JSON(http.StatusOK, resp)
}
