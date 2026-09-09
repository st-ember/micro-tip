package payment

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (ph *PaymentHandler) HandleOrderStatus(c *gin.Context) {
	merchantTradeNo := c.Param("merchant_trade_no")

	stat, err := ph.statusCheckUsecase.Execute(c, merchantTradeNo)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": stat})
}
