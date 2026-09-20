package ecpay_test

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/st-ember/microtip/internal/adpt/driven/hash/ecpay"
)

func TestECPayHasher_GenerateCheckMacVal(t *testing.T) {
	t.Run("Self-Consistent Hashing and Verification Pipeline - SHA256", func(t *testing.T) {
		// Test parameters
		testParams := map[string]string{
			"MerchantID":        "2000132",
			"MerchantTradeNo":   "ecpay20230312153023",
			"MerchantTradeDate": "2023/03/12 15:30:23",
			"PaymentType":       "aio",
			"TotalAmount":       "1000",
			"TradeDesc":         "促銷方案",
			"ItemName":          "Apple iphone 15",
			"ReturnURL":         "https://www.ecpay.com.tw/receive.php",
			"ChoosePayment":     "ALL",
			"EncryptType":       "1",
		}

		hashKey := "5294y06JbISpM5x9"
		hashIV := "v77hoKGq4kWxNNIS"

		// Expected raw URL-encoded lowercase string:
		// We can construct this step-by-step using our proven ECPayUrlEncode.
		rawString := "HashKey=5294y06JbISpM5x9&ChoosePayment=ALL&EncryptType=1&ItemName=Apple iphone 15&MerchantID=2000132&MerchantTradeDate=2023/03/12 15:30:23&MerchantTradeNo=ecpay20230312153023&PaymentType=aio&ReturnURL=https://www.ecpay.com.tw/receive.php&TotalAmount=1000&TradeDesc=促銷方案&HashIV=v77hoKGq4kWxNNIS"
		expectedEncodedString := ecpay.ECPayUrlEncode(rawString)
		
		// The exact expected SHA256 sum of that string:
		h := sha256.Sum256([]byte(expectedEncodedString))
		expectedMAC := strings.ToUpper(fmt.Sprintf("%x", h))

		hasher := ecpay.NewECPayHasher(hashKey, hashIV)
		mac, err := hasher.GenerateCheckMacVal(testParams)
		assert.NoError(t, err)
		assert.Equal(t, expectedMAC, mac)
	})

	t.Run("CheckMacValue key already in map (must be ignored)", func(t *testing.T) {
		testParams := map[string]string{
			"MerchantID":        "2000132",
			"MerchantTradeNo":   "ecpay20230312153023",
			"MerchantTradeDate": "2023/03/12 15:30:23",
			"PaymentType":       "aio",
			"TotalAmount":       "1000",
			"TradeDesc":         "促銷方案",
			"ItemName":          "Apple iphone 15",
			"ReturnURL":         "https://www.ecpay.com.tw/receive.php",
			"ChoosePayment":     "ALL",
			"EncryptType":       "1",
			"CheckMacValue":     "SOME_OLD_MAC_THAT_SHOULD_BE_IGNORED",
		}

		hashKey := "5294y06JbISpM5x9"
		hashIV := "v77hoKGq4kWxNNIS"

		rawString := "HashKey=5294y06JbISpM5x9&ChoosePayment=ALL&EncryptType=1&ItemName=Apple iphone 15&MerchantID=2000132&MerchantTradeDate=2023/03/12 15:30:23&MerchantTradeNo=ecpay20230312153023&PaymentType=aio&ReturnURL=https://www.ecpay.com.tw/receive.php&TotalAmount=1000&TradeDesc=促銷方案&HashIV=v77hoKGq4kWxNNIS"
		expectedEncodedString := ecpay.ECPayUrlEncode(rawString)
		
		h := sha256.Sum256([]byte(expectedEncodedString))
		expectedMAC := strings.ToUpper(fmt.Sprintf("%x", h))

		hasher := ecpay.NewECPayHasher(hashKey, hashIV)
		mac, err := hasher.GenerateCheckMacVal(testParams)
		assert.NoError(t, err)
		assert.Equal(t, expectedMAC, mac)
	})

	t.Run("Validation case-insensitive comparison", func(t *testing.T) {
		testParams := map[string]string{
			"MerchantID":        "2000132",
			"MerchantTradeNo":   "ecpay20230312153023",
			"MerchantTradeDate": "2023/03/12 15:30:23",
			"PaymentType":       "aio",
			"TotalAmount":       "1000",
			"TradeDesc":         "促銷方案",
			"ItemName":          "Apple iphone 15",
			"ReturnURL":         "https://www.ecpay.com.tw/receive.php",
			"ChoosePayment":     "ALL",
			"EncryptType":       "1",
		}

		hashKey := "5294y06JbISpM5x9"
		hashIV := "v77hoKGq4kWxNNIS"

		hasher := ecpay.NewECPayHasher(hashKey, hashIV)
		mac, err := hasher.GenerateCheckMacVal(testParams)
		assert.NoError(t, err)

		// Validate lowercase MAC
		ok, err := hasher.ValidateCheckMacVal(testParams, strings.ToLower(mac))
		assert.NoError(t, err)
		assert.True(t, ok)

		// Validate uppercase MAC
		ok, err = hasher.ValidateCheckMacVal(testParams, mac)
		assert.NoError(t, err)
		assert.True(t, ok)

		// Invalid MAC
		ok, err = hasher.ValidateCheckMacVal(testParams, "invalid_mac")
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("MD5 Encryption (EncryptType = 0)", func(t *testing.T) {
		testParams := map[string]string{
			"MerchantID":        "2000132",
			"MerchantTradeNo":   "ecpay20230312153023",
			"MerchantTradeDate": "2023/03/12 15:30:23",
			"PaymentType":       "aio",
			"TotalAmount":       "1000",
			"TradeDesc":         "促銷方案",
			"ItemName":          "Apple iphone 15",
			"ReturnURL":         "https://www.ecpay.com.tw/receive.php",
			"ChoosePayment":     "ALL",
			"EncryptType":       "0",
		}

		hashKey := "5294y06JbISpM5x9"
		hashIV := "v77hoKGq4kWxNNIS"

		rawString := "HashKey=5294y06JbISpM5x9&ChoosePayment=ALL&EncryptType=0&ItemName=Apple iphone 15&MerchantID=2000132&MerchantTradeDate=2023/03/12 15:30:23&MerchantTradeNo=ecpay20230312153023&PaymentType=aio&ReturnURL=https://www.ecpay.com.tw/receive.php&TotalAmount=1000&TradeDesc=促銷方案&HashIV=v77hoKGq4kWxNNIS"
		expectedEncodedString := ecpay.ECPayUrlEncode(rawString)

		h := md5.Sum([]byte(expectedEncodedString))
		expectedMAC := strings.ToUpper(fmt.Sprintf("%x", h))

		hasher := ecpay.NewECPayHasher(hashKey, hashIV)
		mac, err := hasher.GenerateCheckMacVal(testParams)
		assert.NoError(t, err)
		assert.Equal(t, expectedMAC, mac)
	})
}
