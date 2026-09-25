package tip

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (th *TipHandler) HandleTipTransfer(c *gin.Context) {
	var req TipTransferReq

	// Decipher JSON
	if err := c.BindJSON(&req); err != nil {
		th.logger.ErrorCtx(c, "invalid json request", err, "handler", "tip_transfer")
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid input format")
		return
	}

	// Check domain rules
	cmd, err := req.ToCmd()
	if err != nil {
		th.logger.ErrorCtx(c, "create tip transfer command", err, "handler", "tip_transfer")
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid input format")
		return
	}

	// Execute usecase
	if err := th.tiptransferUC.Execute(c.Request.Context(), cmd); err != nil {
		th.logger.ErrorCtx(c, "execute usecase", err, "handler", "tip_transfer", "usecase", "tip_trasnfer")
		c.AbortWithStatusJSON(http.StatusInternalServerError, "internal error")
		return
	}

	c.Status(http.StatusOK)
}
