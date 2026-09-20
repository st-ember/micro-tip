package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Ensure we clear the relevant environment variables to test defaults
	envVars := []string{
		"DATABASE_URL", "REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"REDIS_BALANCE_EXP_HOUR", "REDIS_ORDER_EXP_HOUR", "ECPAY_HASH_KEY",
		"ECPAY_HASH_IV", "SPLIT_BASIS_POINTS", "PLATFORM_ID",
		"ECPAY_MERCHANT_ID", "ECPAY_TRADE_DESC", "ECPAY_RETURN_URL",
		"ECPAY_CLIENT_BACK_URL", "ECPAY_PAYMENT_URL", "PORT",
	}
	for _, env := range envVars {
		orig := os.Getenv(env)
		defer func(e, o string) {
			_ = os.Setenv(e, o)
		}(env, orig)
		_ = os.Unsetenv(env)
	}

	cfg := LoadConfig()

	if cfg.DatabaseURL != "postgres://postgres:postgres@localhost:5432/microtip?sslmode=disable" {
		t.Errorf("expected default DatabaseURL, got %s", cfg.DatabaseURL)
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("expected default RedisAddr, got %s", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "" {
		t.Errorf("expected default RedisPassword, got %s", cfg.RedisPassword)
	}
	if cfg.RedisDB != 0 {
		t.Errorf("expected default RedisDB, got %d", cfg.RedisDB)
	}
	if cfg.RedisBalanceExp != int(24*time.Hour) {
		t.Errorf("expected default RedisBalanceExp to be 24h, got %d", cfg.RedisBalanceExp)
	}
	if cfg.RedisOrderExp != int(24*time.Hour) {
		t.Errorf("expected default RedisOrderExp to be 24h, got %d", cfg.RedisOrderExp)
	}
	if cfg.ECPayHashKey != "5294y06JbISpM5x9" {
		t.Errorf("expected default ECPayHashKey, got %s", cfg.ECPayHashKey)
	}
	if cfg.ECPayHashIV != "v77hoKGq4kWxNNIS" {
		t.Errorf("expected default ECPayHashIV, got %s", cfg.ECPayHashIV)
	}
	if cfg.SplitBasisPoints != 1525 {
		t.Errorf("expected default SplitBasisPoints, got %d", cfg.SplitBasisPoints)
	}
	if cfg.PlatformID != "platform-789" {
		t.Errorf("expected default PlatformID, got %s", cfg.PlatformID)
	}
	if cfg.ECPayMerchantID != "2000132" {
		t.Errorf("expected default ECPayMerchantID, got %s", cfg.ECPayMerchantID)
	}
	if cfg.ECPayTradeDesc != "live-tip-topup" {
		t.Errorf("expected default ECPayTradeDesc, got %s", cfg.ECPayTradeDesc)
	}
	if cfg.ECPayReturnURL != "http://localhost:8080/payment/confirmation" {
		t.Errorf("expected default ECPayReturnURL, got %s", cfg.ECPayReturnURL)
	}
	if cfg.ECPayClientBackURL != "http://localhost:8080/payment/status" {
		t.Errorf("expected default ECPayClientBackURL, got %s", cfg.ECPayClientBackURL)
	}
	if cfg.ECPayPaymentURL != "https://payment-stage.ecpay.com.tw/Cashier/AioCheckOut/V5" {
		t.Errorf("expected default ECPayPaymentURL, got %s", cfg.ECPayPaymentURL)
	}
	if cfg.Port != ":8080" {
		t.Errorf("expected default Port, got %s", cfg.Port)
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test_user:pass@test_host:5432/test_db")
	t.Setenv("REDIS_ADDR", "test-redis:6379")
	t.Setenv("REDIS_PASSWORD", "test-pass")
	t.Setenv("REDIS_DB", "3")
	t.Setenv("REDIS_BALANCE_EXP_HOUR", "12")
	t.Setenv("REDIS_ORDER_EXP_HOUR", "6")
	t.Setenv("ECPAY_HASH_KEY", "newkey")
	t.Setenv("ECPAY_HASH_IV", "newiv")
	t.Setenv("SPLIT_BASIS_POINTS", "500")
	t.Setenv("PLATFORM_ID", "platform-test")
	t.Setenv("ECPAY_MERCHANT_ID", "999")
	t.Setenv("ECPAY_TRADE_DESC", "test-desc")
	t.Setenv("ECPAY_RETURN_URL", "http://test/return")
	t.Setenv("ECPAY_CLIENT_BACK_URL", "http://test/back")
	t.Setenv("ECPAY_PAYMENT_URL", "https://test/pay")
	t.Setenv("PORT", "9090")

	cfg := LoadConfig()

	if cfg.DatabaseURL != "postgres://test_user:pass@test_host:5432/test_db" {
		t.Errorf("expected overridden DatabaseURL, got %s", cfg.DatabaseURL)
	}
	if cfg.RedisAddr != "test-redis:6379" {
		t.Errorf("expected overridden RedisAddr, got %s", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "test-pass" {
		t.Errorf("expected overridden RedisPassword, got %s", cfg.RedisPassword)
	}
	if cfg.RedisDB != 3 {
		t.Errorf("expected overridden RedisDB, got %d", cfg.RedisDB)
	}
	if cfg.RedisBalanceExp != int(12*time.Hour) {
		t.Errorf("expected overridden RedisBalanceExp, got %d", cfg.RedisBalanceExp)
	}
	if cfg.RedisOrderExp != int(6*time.Hour) {
		t.Errorf("expected overridden RedisOrderExp, got %d", cfg.RedisOrderExp)
	}
	if cfg.ECPayHashKey != "newkey" {
		t.Errorf("expected overridden ECPayHashKey, got %s", cfg.ECPayHashKey)
	}
	if cfg.ECPayHashIV != "newiv" {
		t.Errorf("expected overridden ECPayHashIV, got %s", cfg.ECPayHashIV)
	}
	if cfg.SplitBasisPoints != 500 {
		t.Errorf("expected overridden SplitBasisPoints, got %d", cfg.SplitBasisPoints)
	}
	if cfg.PlatformID != "platform-test" {
		t.Errorf("expected overridden PlatformID, got %s", cfg.PlatformID)
	}
	if cfg.ECPayMerchantID != "999" {
		t.Errorf("expected overridden ECPayMerchantID, got %s", cfg.ECPayMerchantID)
	}
	if cfg.ECPayTradeDesc != "test-desc" {
		t.Errorf("expected overridden ECPayTradeDesc, got %s", cfg.ECPayTradeDesc)
	}
	if cfg.ECPayReturnURL != "http://test/return" {
		t.Errorf("expected overridden ECPayReturnURL, got %s", cfg.ECPayReturnURL)
	}
	if cfg.ECPayClientBackURL != "http://test/back" {
		t.Errorf("expected overridden ECPayClientBackURL, got %s", cfg.ECPayClientBackURL)
	}
	if cfg.ECPayPaymentURL != "https://test/pay" {
		t.Errorf("expected overridden ECPayPaymentURL, got %s", cfg.ECPayPaymentURL)
	}
	if cfg.Port != ":9090" {
		t.Errorf("expected overridden Port with colon prepended, got %s", cfg.Port)
	}
}

func TestLoadConfig_PortWithColon(t *testing.T) {
	t.Setenv("PORT", ":9595")

	cfg := LoadConfig()
	if cfg.Port != ":9595" {
		t.Errorf("expected overridden Port with colon unchanged, got %s", cfg.Port)
	}
}
