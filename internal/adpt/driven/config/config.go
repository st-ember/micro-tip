package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL        string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	RedisBalanceExp    int
	RedisOrderExp      int
	ECPayHashKey       string
	ECPayHashIV        string
	SplitBasisPoints   int64
	PlatformID         string
	ECPayMerchantID    string
	ECPayTradeDesc     string
	ECPayReturnURL     string
	ECPayClientBackURL string
	ECPayPaymentURL    string
	Port               string
}

func LoadConfig() *Config {
	redisBalanceExpHours := getEnvInt("REDIS_BALANCE_EXP_HOUR", 24)
	redisOrderExpHours := getEnvInt("REDIS_ORDER_EXP_HOUR", 24)

	port := getEnv("PORT", ":8080")
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return &Config{
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/microtip?sslmode=disable"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("REDIS_DB", 0),
		RedisBalanceExp:    int(time.Duration(redisBalanceExpHours) * time.Hour),
		RedisOrderExp:      int(time.Duration(redisOrderExpHours) * time.Hour),
		ECPayHashKey:       getEnv("ECPAY_HASH_KEY", "5294y06JbISpM5x9"),
		ECPayHashIV:        getEnv("ECPAY_HASH_IV", "v77hoKGq4kWxNNIS"),
		SplitBasisPoints:   getEnvInt64("SPLIT_BASIS_POINTS", 1525),
		PlatformID:         getEnv("PLATFORM_ID", "platform-789"),
		ECPayMerchantID:    getEnv("ECPAY_MERCHANT_ID", "2000132"),
		ECPayTradeDesc:     getEnv("ECPAY_TRADE_DESC", "live-tip-topup"),
		ECPayReturnURL:     getEnv("ECPAY_RETURN_URL", "http://localhost:8080/payment/confirmation"),
		ECPayClientBackURL: getEnv("ECPAY_CLIENT_BACK_URL", "http://localhost:8080/payment/status"),
		ECPayPaymentURL:    getEnv("ECPAY_PAYMENT_URL", "https://payment-stage.ecpay.com.tw/Cashier/AioCheckOut/V5"),
		Port:               port,
	}
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return fallback
}
