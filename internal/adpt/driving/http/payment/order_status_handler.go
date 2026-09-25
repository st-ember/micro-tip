package payment

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (ph *PaymentHandler) HandleOrderStatus(c *gin.Context) {
	merchantTradeNo := c.Param("merchant_trade_no")
	if merchantTradeNo == "" {
		ph.logger.ErrorCtx(c, "invalid request", errors.New("empty merchant trade no"))
		c.Status(http.StatusBadRequest)
		return
	}

	stat, err := ph.statusCheckUsecase.Execute(c.Request.Context(), merchantTradeNo)
	if err != nil {
		ph.logger.ErrorCtx(c, "execute usecase", err, "handler", "status_check", "usecase", "status_check")
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": stat})
}
