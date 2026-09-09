package payment_test

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

	"github.com/st-ember/microtip/internal/adpt/driving/http/payment"
	hashMocks "github.com/st-ember/microtip/internal/app/port/hash/mocks"
	"github.com/st-ember/microtip/internal/app/usecase"
	usecaseMocks "github.com/st-ember/microtip/internal/app/usecase/mocks"
)

func TestPaymentHandler_HandleCheckout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		merchantID    = "merchant-123"
		tradeDesc     = "live-tip-topup"
		returnURL     = "https://platform.com/cb"
		clientBackURL = "https://platform.com/back"
		paymentURL    = "https://payment.ecpay.com/aio"
		userID        = "user-123"
		itemName      = "streamer-tip-credit"
		totalAmount   = int64(1500)
		tradeNo       = "unique-trade-no-123"
		tradeDate     = "2026/08/29 18:30:00"
		mockMAC       = "F983A89CEB1194E34747B0E68CEE28A18DFE1CD9BA93C89DAEC9B9C496057DE4"
	)

	// Sample payload
	reqBody := payment.CheckoutReq{
		ItemName:    itemName,
		TotalAmount: totalAmount,
		UserID:      userID,
	}

	t.Run("Success - parses request, executes checkout, and returns correct signed ECPay payload", func(t *testing.T) {
		checkoutMock := usecaseMocks.NewMockCheckoutUsecase(t)
		failMock := usecaseMocks.NewMockFailTopupUsecase(t)
		confirmMock := usecaseMocks.NewMockConfirmationUsecase(t)
		statusMock := usecaseMocks.NewMockStatusCheckUsecase(t)
		hasherMock := hashMocks.NewMockHasher(t)

		handler := payment.NewPaymentHandler(
			merchantID, tradeDesc, returnURL, clientBackURL, paymentURL,
			checkoutMock, failMock, confirmMock, statusMock, hasherMock,
		)

		r := gin.New()
		r.POST("/checkout", handler.HandleCheckout)

		// Set up mock usecase expectation
		checkoutMock.EXPECT().
			Execute(mock.Anything, userID, totalAmount).
			Return(&usecase.CheckoutResult{
				MerchantTradeNo:   tradeNo,
				MerchantTradeDate: tradeDate,
			}, nil).
			Once()

		// Set up mock hasher expectation
		hasherMock.EXPECT().
			GenerateCheckMacVal(mock.MatchedBy(func(params map[string]string) bool {
				return params["MerchantID"] == merchantID &&
					params["MerchantTradeNo"] == tradeNo &&
					params["MerchantTradeDate"] == tradeDate &&
					params["TotalAmount"] == "1500" &&
					params["ItemName"] == itemName &&
					params["ReturnURL"] == returnURL &&
					params["ClientBackURL"] == clientBackURL
			})).
			Return(mockMAC, nil).
			Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/checkout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp payment.ECPayCheckoutResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)

		assert.Equal(t, paymentURL, resp.PaymentURL)
		assert.Equal(t, merchantID, resp.Params.MerchantID)
		assert.Equal(t, tradeNo, resp.Params.MerchantTradeNo)
		assert.Equal(t, tradeDate, resp.Params.MerchantTradeDate)
		assert.Equal(t, totalAmount, resp.Params.TotalAmount)
		assert.Equal(t, itemName, resp.Params.ItemName)
		assert.Equal(t, mockMAC, resp.Params.CheckMacValue)
	})

	t.Run("Error - invalid JSON body", func(t *testing.T) {
		checkoutMock := usecaseMocks.NewMockCheckoutUsecase(t)
		failMock := usecaseMocks.NewMockFailTopupUsecase(t)
		confirmMock := usecaseMocks.NewMockConfirmationUsecase(t)
		statusMock := usecaseMocks.NewMockStatusCheckUsecase(t)
		hasherMock := hashMocks.NewMockHasher(t)

		handler := payment.NewPaymentHandler(
			merchantID, tradeDesc, returnURL, clientBackURL, paymentURL,
			checkoutMock, failMock, confirmMock, statusMock, hasherMock,
		)

		r := gin.New()
		r.POST("/checkout", handler.HandleCheckout)

		req, _ := http.NewRequest(http.MethodPost, "/checkout", bytes.NewBufferString("{invalid-json"))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid input format")
	})

	t.Run("Error - checkout usecase fails", func(t *testing.T) {
		checkoutMock := usecaseMocks.NewMockCheckoutUsecase(t)
		failMock := usecaseMocks.NewMockFailTopupUsecase(t)
		confirmMock := usecaseMocks.NewMockConfirmationUsecase(t)
		statusMock := usecaseMocks.NewMockStatusCheckUsecase(t)
		hasherMock := hashMocks.NewMockHasher(t)

		handler := payment.NewPaymentHandler(
			merchantID, tradeDesc, returnURL, clientBackURL, paymentURL,
			checkoutMock, failMock, confirmMock, statusMock, hasherMock,
		)

		r := gin.New()
		r.POST("/checkout", handler.HandleCheckout)

		checkoutMock.EXPECT().
			Execute(mock.Anything, userID, totalAmount).
			Return(nil, errors.New("database connection down")).
			Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/checkout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "internal server error")
	})

	t.Run("Error - hasher generation fails", func(t *testing.T) {
		checkoutMock := usecaseMocks.NewMockCheckoutUsecase(t)
		failMock := usecaseMocks.NewMockFailTopupUsecase(t)
		confirmMock := usecaseMocks.NewMockConfirmationUsecase(t)
		statusMock := usecaseMocks.NewMockStatusCheckUsecase(t)
		hasherMock := hashMocks.NewMockHasher(t)

		handler := payment.NewPaymentHandler(
			merchantID, tradeDesc, returnURL, clientBackURL, paymentURL,
			checkoutMock, failMock, confirmMock, statusMock, hasherMock,
		)

		r := gin.New()
		r.POST("/checkout", handler.HandleCheckout)

		checkoutMock.EXPECT().
			Execute(mock.Anything, userID, totalAmount).
			Return(&usecase.CheckoutResult{
				MerchantTradeNo:   tradeNo,
				MerchantTradeDate: tradeDate,
			}, nil).
			Once()

		hasherMock.EXPECT().
			GenerateCheckMacVal(mock.Anything).
			Return("", errors.New("failed to hash keys")).
			Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/checkout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to generate payment signature")
	})
}
