package payment

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type CheckoutReq struct {
	ItemName    string `json:"item_name" binding:"required"`
	TotalAmount int64  `json:"total_amount" binding:"required"`
	UserID      string `json:"user_id" binding:"required"`
}

type ECPayCheckoutResponse struct {
	PaymentURL string             `json:"payment_url"`
	Params     ECPayPaymentParams `json:"params"`
}

type ECPayPaymentParams struct {
	MerchantID        string `form:"MerchantID" json:"MerchantID" binding:"required"`
	MerchantTradeNo   string `form:"MerchantTradeNo" json:"MerchantTradeNo" binding:"required"`
	MerchantTradeDate string `form:"MerchantTradeDate" json:"MerchantTradeDate" binding:"required"`
	PaymentType       string `form:"PaymentType" json:"PaymentType" binding:"required"`
	TotalAmount       int64  `form:"TotalAmount" json:"TotalAmount" binding:"required"`
	TradeDesc         string `form:"TradeDesc" json:"TradeDesc" binding:"required"`
	ItemName          string `form:"ItemName" json:"ItemName" binding:"required"`
	ReturnURL         string `form:"ReturnURL" json:"ReturnURL" binding:"required"`
	ChoosePayment     string `form:"ChoosePayment" json:"ChoosePayment" binding:"required"`
	CheckMacValue     string `form:"CheckMacValue" json:"CheckMacValue" binding:"required"`
	EncryptType       int    `form:"EncryptType" json:"EncryptType" binding:"required"`
	ClientBackURL     string `form:"ClientBackURL,omitempty" json:"ClientBackURL,omitempty"`
}

func (p *ECPayPaymentParams) ToMap() map[string]string {
	result := make(map[string]string)
	v := reflect.ValueOf(p).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		// Skip CheckMacValue because it's calculated *from* the other parameters
		if fieldType.Name == "CheckMacValue" {
			continue
		}

		// Get the form tag name, fallback to field name if tag is missing
		tag := fieldType.Tag.Get("form")
		if tag == "" {
			tag = fieldType.Tag.Get("json")
		}
		if tag == "" || tag == "-" {
			continue
		}
		// Handle tags with options like "ClientBackURL,omitempty"
		if idx := strings.Index(tag, ","); idx != -1 {
			tag = tag[:idx]
		}

		var strVal string
		switch fieldVal.Kind() {
		case reflect.String:
			strVal = fieldVal.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			strVal = strconv.FormatInt(fieldVal.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			strVal = strconv.FormatUint(fieldVal.Uint(), 10)
		case reflect.Bool:
			strVal = strconv.FormatBool(fieldVal.Bool())
		default:
			strVal = fmt.Sprintf("%v", fieldVal.Interface())
		}

		// Omit empty strings (or omitted optional fields)
		if strVal != "" {
			result[tag] = strVal
		}
	}

	return result
}

type ECPayReturnForm struct {
	MerchantID      string `form:"MerchantID" binding:"required"`
	MerchantTradeNo string `form:"MerchantTradeNo" binding:"required"`
	RtnCode         int    `form:"RtnCode"`
	RtnMsg          string `form:"RtnMsg"`
	TradeNo         string `form:"TradeNo"`
	TradeAmt        int    `form:"TradeAmt"`
	PaymentDate     string `form:"PaymentDate"`
	PaymentType     string `form:"PaymentType"`
	CheckMacValue   string `form:"CheckMacValue" binding:"required"`
}

func (f *ECPayReturnForm) ToMap() map[string]string {
	result := make(map[string]string)
	v := reflect.ValueOf(f).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		// CheckMacValue is generated *from* the other fields, so exclude it
		if fieldType.Name == "CheckMacValue" {
			continue
		}

		tag := fieldType.Tag.Get("form")
		if tag == "" || tag == "-" {
			continue
		}

		var strVal string
		switch fieldVal.Kind() {
		case reflect.String:
			strVal = fieldVal.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// Ensures integers like RtnCode (even if 0) and TradeAmt convert correctly to strings
			strVal = strconv.FormatInt(fieldVal.Int(), 10)
		default:
			strVal = fmt.Sprintf("%v", fieldVal.Interface())
		}

		// Only include non-empty values (or "0" for numeric codes)
		if strVal != "" {
			result[tag] = strVal
		}
	}

	return result
}

const (
	PaymentTypeAIO      = "aio"
	ChoosePaymentCredit = "Credit"
	EncryptTypeSHA256   = 1
)
