package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/st-ember/microtip/internal/domain"
)

func TestNewRedisCache(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		cache, err := NewRedisCache(ctx, mr.Addr(), "", 0, 10, 10)
		assert.NoError(t, err)
		assert.NotNil(t, cache)
	})

	t.Run("connection error", func(t *testing.T) {
		cache, err := NewRedisCache(ctx, "localhost:99999", "", 0, 10, 10)
		assert.Error(t, err)
		assert.Nil(t, cache)
	})
}

func TestReadBalance(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := &RedisCache{
		Client:     rdb,
		balanceExp: 60,
		orderExp:   60,
	}

	t.Run("success", func(t *testing.T) {
		userID := "user-1"
		bc := BalanceCache{
			UserID:         userID,
			CurrentBalance: 1000,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		data, err := json.Marshal(bc)
		require.NoError(t, err)

		err = mr.Set(userID, string(data))
		require.NoError(t, err)

		balance, err := cache.ReadBalance(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, balance)
		assert.Equal(t, userID, balance.UserID)
		assert.Equal(t, int64(1000), balance.CurrentBalance)
	})

	t.Run("key not found", func(t *testing.T) {
		balance, err := cache.ReadBalance(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, balance)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("invalid json", func(t *testing.T) {
		userID := "user-invalid"
		err = mr.Set(userID, "not-json")
		require.NoError(t, err)

		balance, err := cache.ReadBalance(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, balance)
		assert.Contains(t, err.Error(), "deserialize balance")
	})
}

func TestSaveBalance(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := &RedisCache{
		Client:     rdb,
		balanceExp: int(time.Minute),
		orderExp:   int(time.Minute),
	}

	balance := &domain.Balance{
		UserID:         "user-1",
		CurrentBalance: 500,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	err = cache.SaveBalance(ctx, balance)
	assert.NoError(t, err)

	val, err := mr.Get(balance.UserID)
	assert.NoError(t, err)

	var bc BalanceCache
	err = json.Unmarshal([]byte(val), &bc)
	assert.NoError(t, err)
	assert.Equal(t, balance.UserID, bc.UserID)
	assert.Equal(t, balance.CurrentBalance, bc.CurrentBalance)
}

func TestReadOrder(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := &RedisCache{
		Client:     rdb,
		balanceExp: 60,
		orderExp:   60,
	}

	t.Run("success", func(t *testing.T) {
		merchantTradeNo := "trade-123"
		oc := OrderCache{
			UserID:         "user-1",
			MerchantradeNo: merchantTradeNo,
			Amount:         150,
			Status:         domain.OrderStatusPending,
			CreatedAt:      time.Now().UTC(),
		}
		data, err := json.Marshal(oc)
		require.NoError(t, err)

		err = mr.Set(merchantTradeNo, string(data))
		require.NoError(t, err)

		order, err := cache.ReadOrder(ctx, merchantTradeNo)
		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, merchantTradeNo, order.MerchantradeNo)
		assert.Equal(t, domain.OrderStatusPending, order.Status)
	})

	t.Run("key not found", func(t *testing.T) {
		order, err := cache.ReadOrder(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("invalid json", func(t *testing.T) {
		tradeNo := "trade-invalid"
		err = mr.Set(tradeNo, "not-json")
		require.NoError(t, err)

		order, err := cache.ReadOrder(ctx, tradeNo)
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "deserialize order")
	})
}

func TestSaveOrder(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := &RedisCache{
		Client:     rdb,
		balanceExp: int(time.Minute),
		orderExp:   int(time.Minute),
	}

	order := &domain.Order{
		UserID:         "user-1",
		MerchantradeNo: "trade-123",
		Amount:         200,
		Status:         domain.OrderStatusPending,
		CreatedAt:      time.Now().UTC(),
	}

	err = cache.SaveOrder(ctx, order)
	assert.NoError(t, err)

	val, err := mr.Get(order.MerchantradeNo)
	assert.NoError(t, err)

	var oc OrderCache
	err = json.Unmarshal([]byte(val), &oc)
	assert.NoError(t, err)
	assert.Equal(t, order.MerchantradeNo, oc.MerchantradeNo)
	assert.Equal(t, order.Amount, oc.Amount)
}

func TestInvalidateKey(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := &RedisCache{
		Client:     rdb,
		balanceExp: 60,
		orderExp:   60,
	}

	key := "test-key"
	err = mr.Set(key, "some-value")
	require.NoError(t, err)

	err = cache.InvalidateKey(ctx, key)
	assert.NoError(t, err)

	assert.False(t, mr.Exists(key))
}
