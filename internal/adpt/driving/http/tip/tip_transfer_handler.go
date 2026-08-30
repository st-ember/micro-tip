package tip

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (th *TipHandler) HandleTipTransfer(c *gin.Context) {
	var req TipTransferReq

	// Decipher JSON
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid input format")
		return
	}

	// Check domain rules
	cmd, err := req.ToCmd()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid input format")
		return
	}

	// Execute usecase
	if err := th.tiptransferUC.Execute(c, cmd); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, "internal error")
		return
	}

	c.Status(http.StatusOK)
}
