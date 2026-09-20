package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/st-ember/microtip/internal/app/port/cache"
	"github.com/st-ember/microtip/internal/domain"
)

type RedisCache struct {
	Client     *redis.Client
	balanceExp int
	orderExp   int
}

func NewRedisCache(ctx context.Context, addr, password string, db int, balanceExp, orderxp int) (cache.Cache, error) {
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
		Client:     rdb,
		balanceExp: balanceExp,
		orderExp:   orderxp,
	}, nil
}

func (c *RedisCache) ReadBalance(ctx context.Context, key string) (*domain.Balance, error) {
	val, err := c.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("key %s does not exist", key)
	} else if err != nil {
		return nil, fmt.Errorf("key %s: %w", key, err)
	}

	var bc BalanceCache
	err = json.Unmarshal([]byte(val), &bc)
	if err != nil {
		return nil, fmt.Errorf("deserialize balance for key %s: %w", key, err)
	}

	return bc.ToDomain(), nil
}

func (c *RedisCache) SaveBalance(ctx context.Context, balance *domain.Balance) error {
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

	err = c.Client.Set(ctx, balance.UserID, data, time.Duration(c.balanceExp)).Err()
	if err != nil {
		return fmt.Errorf("set balance for key %s: %w", balance.UserID, err)
	}

	return nil
}

func (c *RedisCache) ReadOrder(ctx context.Context, key string) (*domain.Order, error) {
	val, err := c.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("key %s does not exist", key)
	} else if err != nil {
		return nil, fmt.Errorf("key %s: %w", key, err)
	}

	var oc OrderCache
	err = json.Unmarshal([]byte(val), &oc)
	if err != nil {
		return nil, fmt.Errorf("deserialize order for key %s: %w", key, err)
	}

	return oc.ToDomain(), nil
}

func (c *RedisCache) SaveOrder(ctx context.Context, order *domain.Order) error {
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

	err = c.Client.Set(ctx, order.MerchantradeNo, data, time.Duration(c.orderExp)).Err()
	if err != nil {
		return fmt.Errorf("set balance for key %s: %w", order.MerchantradeNo, err)
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
