package payment

import (
	"github.com/st-ember/microtip/internal/app/port/hash"
	"github.com/st-ember/microtip/internal/app/usecase"
)

type PaymentHandler struct {
	merchantID          string
	tradeDesc           string
	returnURL           string
	clientBackURL       string
	paymentURL          string
	checkoutUC          usecase.CheckoutUsecase
	failTopupUC         usecase.FailTopupUsecase
	confirmationUsecase usecase.ConfirmationUsecase
	statusCheckUsecase  usecase.StatusCheckUsecase
	hasher              hash.Hasher
}

func NewPaymentHandler(
	merchantID string,
	tradeDesc string,
	returnURL string,
	clientBackURL string,
	paymentURL string,
	checkoutUC usecase.CheckoutUsecase,
	failTopupUC usecase.FailTopupUsecase,
	confirmationUsecase usecase.ConfirmationUsecase,
	statusCheckUsecase usecase.StatusCheckUsecase,
	hasher hash.Hasher,
) *PaymentHandler {
	return &PaymentHandler{
		merchantID:          merchantID,
		tradeDesc:           tradeDesc,
		returnURL:           returnURL,
		clientBackURL:       clientBackURL,
		paymentURL:          paymentURL,
		checkoutUC:          checkoutUC,
		failTopupUC:         failTopupUC,
		confirmationUsecase: confirmationUsecase,
		statusCheckUsecase:  statusCheckUsecase,
		hasher:              hasher,
	}
}
