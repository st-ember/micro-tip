package tip_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/st-ember/microtip/internal/adpt/driving/http/tip"
	usecaseMocks "github.com/st-ember/microtip/internal/app/usecase/mocks"
)

func TestTipTransferHandler(t *testing.T) {
	// Set Gin to test mode to avoid verbose logs in output
	gin.SetMode(gin.TestMode)

	const (
		senderID  = "sender-123"
		creatorID = "creator-456"
		idpKey    = "idp-123"
		amount    = int64(100)
	)

	// Valid payload
	validPayload := tip.TipTransferReq{
		SenderID:       senderID,
		CreatorID:      creatorID,
		IdempotencyKey: idpKey,
		Amount:         amount,
	}

	t.Run("Success - valid payload executes successfully", func(t *testing.T) {
		ucMock := usecaseMocks.NewMockTipTransferUsecase(t)
		handler := tip.NewTipHandler(ucMock)

		// Setup route
		r := gin.New()
		r.POST("/tip-transfer", handler.HandleTipTransfer)

		// Mock expectations
		ucMock.EXPECT().
			Execute(mock.Anything, mock.MatchedBy(func(cmd interface{}) bool {
				return true
			})).
			Return(nil).
			Once()

		// Perform request
		body, _ := json.Marshal(validPayload)
		req, _ := http.NewRequest(http.MethodPost, "/tip-transfer", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Error - invalid JSON format", func(t *testing.T) {
		ucMock := usecaseMocks.NewMockTipTransferUsecase(t)
		handler := tip.NewTipHandler(ucMock)

		r := gin.New()
		r.POST("/tip-transfer", handler.HandleTipTransfer)

		// Perform request with malformed JSON
		req, _ := http.NewRequest(http.MethodPost, "/tip-transfer", bytes.NewBufferString(`{invalid-json`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid input format")
	})

	t.Run("Error - missing required field", func(t *testing.T) {
		ucMock := usecaseMocks.NewMockTipTransferUsecase(t)
		handler := tip.NewTipHandler(ucMock)

		r := gin.New()
		r.POST("/tip-transfer", handler.HandleTipTransfer)

		// Payload missing CreatorID
		invalidPayload := tip.TipTransferReq{
			SenderID:       senderID,
			IdempotencyKey: idpKey,
			Amount:         amount,
		}

		body, _ := json.Marshal(invalidPayload)
		req, _ := http.NewRequest(http.MethodPost, "/tip-transfer", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid input format")
	})

	t.Run("Error - domain rule validation failure (tip self)", func(t *testing.T) {
		ucMock := usecaseMocks.NewMockTipTransferUsecase(t)
		handler := tip.NewTipHandler(ucMock)

		r := gin.New()
		r.POST("/tip-transfer", handler.HandleTipTransfer)

		// SenderID == CreatorID
		selfTippingPayload := tip.TipTransferReq{
			SenderID:       senderID,
			CreatorID:      senderID,
			IdempotencyKey: idpKey,
			Amount:         amount,
		}

		body, _ := json.Marshal(selfTippingPayload)
		req, _ := http.NewRequest(http.MethodPost, "/tip-transfer", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid input format")
	})

	t.Run("Error - usecase execution failure", func(t *testing.T) {
		ucMock := usecaseMocks.NewMockTipTransferUsecase(t)
		handler := tip.NewTipHandler(ucMock)

		r := gin.New()
		r.POST("/tip-transfer", handler.HandleTipTransfer)

		// Mock execute fails
		ucMock.EXPECT().
			Execute(mock.Anything, mock.Anything).
			Return(errors.New("db transfer error")).
			Once()

		body, _ := json.Marshal(validPayload)
		req, _ := http.NewRequest(http.MethodPost, "/tip-transfer", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "internal error")
	})
}
