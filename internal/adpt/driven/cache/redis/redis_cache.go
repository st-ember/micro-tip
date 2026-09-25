package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/st-ember/microtip/internal/app/port/cache"
	"github.com/st-ember/microtip/internal/app/port/metrics"
	"github.com/st-ember/microtip/internal/domain"
)

type RedisCache struct {
	Client        *redis.Client
	metrics       metrics.Metrics
	balanceExpSec int
	orderExpSec   int
}

func NewRedisCache(
	ctx context.Context, metrics metrics.Metrics,
	addr, password string, db int, balanceExpSec, orderExpSec int,
) (cache.Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}

	return &RedisCache{
		Client:        rdb,
		metrics:       metrics,
		balanceExpSec: balanceExpSec,
		orderExpSec:   orderExpSec,
	}, nil
}

func (c *RedisCache) ReadBalance(ctx context.Context, key string) (*domain.Balance, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.metrics.ObserveCacheDuration("read_balance", duration)
	}()

	val, err := c.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		c.metrics.IncCacheRead("balance", "miss")
		return nil, fmt.Errorf("key %s does not exist", key)
	} else if err != nil {
		c.metrics.IncCacheRead("balance", "error")
		return nil, fmt.Errorf("key %s: %w", key, err)
	}

	var bc BalanceCache
	err = json.Unmarshal([]byte(val), &bc)
	if err != nil {
		return nil, fmt.Errorf("deserialize balance for key %s: %w", key, err)
	}

	c.metrics.IncCacheRead("balance", "hit")
	return bc.ToDomain(), nil
}

func (c *RedisCache) SaveBalance(ctx context.Context, balance *domain.Balance) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.metrics.ObserveCacheDuration("save_balance", duration)
	}()

	bc := BalanceCache{
		UserID:         balance.UserID,
		CurrentBalance: balance.CurrentBalance,
		CreatedAt:      balance.CreatedAt,
		UpdatedAt:      balance.UpdatedAt,
	}
	data, err := json.Marshal(bc)
	if err != nil {
		return fmt.Errorf("serialize balance for key %s: %w", balance.UserID, err)
	}

	err = c.Client.Set(ctx, balance.UserID, data, time.Duration(c.balanceExpSec)*time.Second).Err()
	if err != nil {
		return fmt.Errorf("set balance for key %s: %w", balance.UserID, err)
	}

	return nil
}

func (c *RedisCache) ReadOrder(ctx context.Context, key string) (*domain.Order, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.metrics.ObserveCacheDuration("read_order", duration)
	}()

	val, err := c.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		c.metrics.IncCacheRead("order", "miss")
		return nil, fmt.Errorf("key %s does not exist", key)
	} else if err != nil {
		c.metrics.IncCacheRead("order", "error")
		return nil, fmt.Errorf("key %s: %w", key, err)
	}

	var oc OrderCache
	err = json.Unmarshal([]byte(val), &oc)
	if err != nil {
		return nil, fmt.Errorf("deserialize order for key %s: %w", key, err)
	}

	c.metrics.IncCacheRead("order", "hit")
	return oc.ToDomain(), nil
}

func (c *RedisCache) SaveOrder(ctx context.Context, order *domain.Order) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.metrics.ObserveCacheDuration("save_order", duration)
	}()

	oc := OrderCache{
		UserID:         order.UserID,
		MerchantradeNo: order.MerchantradeNo,
		Amount:         order.Amount,
		Status:         order.Status,
		CreatedAt:      order.CreatedAt,
	}
	data, err := json.Marshal(oc)
	if err != nil {
		return fmt.Errorf("serialize order for key %s: %w", order.MerchantradeNo, err)
	}

	err = c.Client.Set(ctx, order.MerchantradeNo, data, time.Duration(c.orderExpSec)*time.Second).Err()
	if err != nil {
		return fmt.Errorf("set order for key %s: %w", order.MerchantradeNo, err)
	}

	return nil
}

func (c *RedisCache) InvalidateKey(ctx context.Context, key string) error {
	err := c.Client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("invalidate key %s: %w", key, err)
	}

	return nil
}
