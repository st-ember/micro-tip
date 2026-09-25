package payment_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/st-ember/microtip/internal/adpt/driving/http/payment"
	hashMocks "github.com/st-ember/microtip/internal/app/port/hash/mocks"
	logMocks "github.com/st-ember/microtip/internal/app/port/log/mocks"
	usecaseMocks "github.com/st-ember/microtip/internal/app/usecase/mocks"
)

func TestNewPaymentHandler(t *testing.T) {
	t.Run("NewPaymentHandler - initializes correctly", func(t *testing.T) {
		merchantID := "merchant-123"
		tradeDesc := "live-tip-topup"
		returnURL := "https://platform.com/cb"
		clientBackURL := "https://platform.com/back"
		paymentURL := "https://payment.ecpay.com/aio"

		checkoutMock := usecaseMocks.NewMockCheckoutUsecase(t)
		failMock := usecaseMocks.NewMockFailTopupUsecase(t)
		confirmMock := usecaseMocks.NewMockConfirmationUsecase(t)
		statusMock := usecaseMocks.NewMockStatusCheckUsecase(t)
		hasherMock := hashMocks.NewMockHasher(t)
		mockLogger := logMocks.NewMockLogger(t)

		handler := payment.NewPaymentHandler(
			merchantID, tradeDesc, returnURL, clientBackURL, paymentURL,
			checkoutMock, failMock, confirmMock, statusMock, hasherMock,
			mockLogger,
		)
		assert.NotNil(t, handler)
	})
}
