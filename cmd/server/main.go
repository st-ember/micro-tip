package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/st-ember/microtip/internal/adpt/driven/cache/redis"
	"github.com/st-ember/microtip/internal/adpt/driven/config"
	"github.com/st-ember/microtip/internal/adpt/driven/hash/ecpay"
	"github.com/st-ember/microtip/internal/adpt/driven/log/slogger"
	prometheusmetrics "github.com/st-ember/microtip/internal/adpt/driven/metrics/prometheus"
	"github.com/st-ember/microtip/internal/adpt/driven/repo/postgres"
	drivingHttp "github.com/st-ember/microtip/internal/adpt/driving/http"
	"github.com/st-ember/microtip/internal/app/usecase"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Database Connection
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()

	db, err := postgres.NewDB(dbCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer func() {
		_ = db.Conn.Close()
	}()

	// Initialize metrics and logger
	logger := slogger.NewSlogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	metrics := prometheusmetrics.NewPrometheusMetrics()

	// Redis Cache Connection
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisCancel()

	redisCache, err := redis.NewRedisCache(
		redisCtx, metrics, cfg.RedisAddr, cfg.RedisPassword,
		cfg.RedisDB, cfg.RedisBalanceExp, cfg.RedisOrderExp,
	)
	if err != nil {
		log.Fatalf("Failed to init Redis: %v", err)
	}

	// ECPay Hasher Configuration
	hasher := ecpay.NewECPayHasher(cfg.ECPayHashKey, cfg.ECPayHashIV)

	// Unit of Work Factory & PostgreSQL Repositories
	uowFactory := postgres.NewPostgresUnitOfWorkFactory(db.Conn)
	balanceRepoReadOnly := postgres.NewPostgresBalanceRepo(db.Conn)

	// Usecases
	tipTransferUC := usecase.NewTipTransferUsecase(
		uowFactory,
		redisCache,
		balanceRepoReadOnly,
		metrics,
		logger,
		cfg.SplitBasisPoints,
		cfg.PlatformID,
	)

	checkoutUC := usecase.NewCheckoutUsecase(redisCache, hasher)
	failTopupUC := usecase.NewFailTopupUsecase(redisCache)
	confirmationUC := usecase.NewConfirmationUsecase(uowFactory, redisCache, hasher, logger)
	statusCheckUC := usecase.NewStatusCheckUsecase(redisCache)

	// Router Initialization
	router := drivingHttp.NewRouter(
		tipTransferUC,
		checkoutUC,
		failTopupUC,
		confirmationUC,
		statusCheckUC,
		hasher,
		logger,
		metrics,
		cfg.ECPayMerchantID,
		cfg.ECPayTradeDesc,
		cfg.ECPayReturnURL,
		cfg.ECPayClientBackURL,
		cfg.ECPayPaymentURL,
	)

	log.Printf("Listening and serving HTTP on %s", cfg.Port)
	if err := router.Engine.Run(cfg.Port); err != nil {
		log.Fatalf("Server run failed: %v", err)
	}
}
