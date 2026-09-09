package usecase

type ConfirmationInput struct {
	CheckMacValue       string
	CheckMacValueParams map[string]string
	MerchantTradeNo     string
	Amount              int64
}
