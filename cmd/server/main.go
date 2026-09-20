package main

import (
	"context"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/st-ember/microtip/internal/adpt/driven/cache/redis"
	"github.com/st-ember/microtip/internal/adpt/driven/config"
	"github.com/st-ember/microtip/internal/adpt/driven/hash/ecpay"
	"github.com/st-ember/microtip/internal/adpt/driven/repo/postgres"
	drivingHttp "github.com/st-ember/microtip/internal/adpt/driving/http"
	"github.com/st-ember/microtip/internal/app/usecase"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// 1. Database Connection
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()

	db, err := postgres.NewDB(dbCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer func() {
		_ = db.Conn.Close()
	}()

	// 2. Redis Cache Connection
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisCancel()

	redisCache, err := redis.NewRedisCache(
		redisCtx, cfg.RedisAddr, cfg.RedisPassword,
		cfg.RedisDB, cfg.RedisBalanceExp, cfg.RedisOrderExp,
	)
	if err != nil {
		log.Fatalf("Failed to init Redis: %v", err)
	}

	// 3. ECPay Hasher Configuration
	hasher := ecpay.NewECPayHasher(cfg.ECPayHashKey, cfg.ECPayHashIV)

	// 4. Unit of Work Factory & PostgreSQL Repositories
	uowFactory := postgres.NewPostgresUnitOfWorkFactory(db.Conn)
	balanceRepoReadOnly := postgres.NewPostgresBalanceRepo(db.Conn)

	// 5. Usecases
	tipTransferUC := usecase.NewTipTransferUsecase(
		uowFactory,
		redisCache,
		balanceRepoReadOnly,
		cfg.SplitBasisPoints,
		cfg.PlatformID,
	)

	checkoutUC := usecase.NewCheckoutUsecase(redisCache, hasher)
	failTopupUC := usecase.NewFailTopupUsecase(redisCache)
	confirmationUC := usecase.NewConfirmationUsecase(uowFactory, redisCache, hasher)
	statusCheckUC := usecase.NewStatusCheckUsecase(redisCache)

	// 6. Router Initialization
	router := drivingHttp.NewRouter(
		tipTransferUC,
		checkoutUC,
		failTopupUC,
		confirmationUC,
		statusCheckUC,
		hasher,
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
