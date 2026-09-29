package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	nopmetrics "github.com/st-ember/microtip/internal/adpt/driven/metrics/prometheus/nop"
	"github.com/st-ember/microtip/internal/app/port/metrics"
	"github.com/st-ember/microtip/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type spyMetrics struct {
	cacheReadCalls     map[string]int
	cacheDurationCalls map[string]int
}

func newSpyMetrics() *spyMetrics {
	return &spyMetrics{
		cacheReadCalls:     make(map[string]int),
		cacheDurationCalls: make(map[string]int),
	}
}

func (s *spyMetrics) IncHttpRequest(method, path, status string)                      {}
func (s *spyMetrics) ObserveHttpRequestDuration(method, path string, duration time.Duration) {}
func (s *spyMetrics) IncTip(status metrics.TipStatus)                                 {}
func (s *spyMetrics) ObserveDBTransactionDuration(operation string, duration time.Duration) {}

func (s *spyMetrics) IncCacheRead(cacheName, status string) {
	s.cacheReadCalls[cacheName+"_"+status]++
}

func (s *spyMetrics) ObserveCacheDuration(operation string, duration time.Duration) {
	s.cacheDurationCalls[operation]++
}

func TestNewRedisCache(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		nopMet := nopmetrics.NewNopMetrics()
		cache, err := NewRedisCache(ctx, nopMet, mr.Addr(), "", 0, 10, 10)
		assert.NoError(t, err)
		assert.NotNil(t, cache)
	})

	t.Run("connection error", func(t *testing.T) {
		nopMet := nopmetrics.NewNopMetrics()
		cache, err := NewRedisCache(ctx, nopMet, "localhost:99999", "", 0, 10, 10)
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
	spy := newSpyMetrics()
	cache := &RedisCache{
		Client:        rdb,
		metrics:       spy,
		balanceExpSec: 60,
		orderExpSec:   60,
	}

	t.Run("success and metrics tracked", func(t *testing.T) {
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

		// Verify spy metrics
		assert.Equal(t, 1, spy.cacheReadCalls["balance_hit"])
		assert.Equal(t, 1, spy.cacheDurationCalls["read_balance"])
	})

	t.Run("key not found and metrics tracked", func(t *testing.T) {
		balance, err := cache.ReadBalance(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, balance)
		assert.Contains(t, err.Error(), "does not exist")

		// Verify spy metrics
		assert.Equal(t, 1, spy.cacheReadCalls["balance_miss"])
		assert.Equal(t, 2, spy.cacheDurationCalls["read_balance"])
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
	spy := newSpyMetrics()
	cache := &RedisCache{
		Client:        rdb,
		metrics:       spy,
		balanceExpSec: int(time.Minute),
		orderExpSec:   int(time.Minute),
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

	// Verify spy metrics
	assert.Equal(t, 1, spy.cacheDurationCalls["save_balance"])
}

func TestReadOrder(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	spy := newSpyMetrics()
	cache := &RedisCache{
		Client:        rdb,
		metrics:       spy,
		balanceExpSec: 60,
		orderExpSec:   60,
	}

	t.Run("success and metrics tracked", func(t *testing.T) {
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

		// Verify spy metrics
		assert.Equal(t, 1, spy.cacheReadCalls["order_hit"])
		assert.Equal(t, 1, spy.cacheDurationCalls["read_order"])
	})

	t.Run("key not found and metrics tracked", func(t *testing.T) {
		order, err := cache.ReadOrder(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Contains(t, err.Error(), "does not exist")

		// Verify spy metrics
		assert.Equal(t, 1, spy.cacheReadCalls["order_miss"])
		assert.Equal(t, 2, spy.cacheDurationCalls["read_order"])
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
	spy := newSpyMetrics()
	cache := &RedisCache{
		Client:        rdb,
		metrics:       spy,
		balanceExpSec: int(time.Minute),
		orderExpSec:   int(time.Minute),
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

	// Verify spy metrics
	assert.Equal(t, 1, spy.cacheDurationCalls["save_order"])
}

func TestInvalidateKey(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := &RedisCache{
		Client:        rdb,
		metrics:       nopmetrics.NewNopMetrics(),
		balanceExpSec: 60,
		orderExpSec:   60,
	}

	key := "test-key"
	err = mr.Set(key, "some-value")
	require.NoError(t, err)

	err = cache.InvalidateKey(ctx, key)
	assert.NoError(t, err)

	assert.False(t, mr.Exists(key))
}
