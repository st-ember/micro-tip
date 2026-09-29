package prometheusmetrics_test

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prometheusmetrics "github.com/st-ember/microtip/internal/adpt/driven/metrics/prometheus"
	"github.com/st-ember/microtip/internal/app/port/metrics"
)

func TestNewPrometheusMetrics(t *testing.T) {
	// Open a dummy DB connection (no real connection is made until queried, which NewDBStatsCollector doesn't do)
	db, err := sql.Open("postgres", "postgres://localhost:5432/dummy?sslmode=disable")
	require.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	reg := prometheus.NewRegistry()
	pm := prometheusmetrics.NewPrometheusMetrics(reg, db)
	require.NotNil(t, pm)

	// Trigger API metrics to test registration and recording
	t.Run("HTTP Request Metric", func(t *testing.T) {
		pm.IncHttpRequest("POST", "/tip-transfer", "200")
		pm.ObserveHttpRequestDuration("POST", "/tip-transfer", 150*time.Millisecond)

		mfs, err := reg.Gather()
		assert.NoError(t, err)

		foundTotal := false
		foundDuration := false

		for _, mf := range mfs {
			if mf.GetName() == "micro_tip_http_requests_total" {
				foundTotal = true
				metric := mf.GetMetric()[0]
				// Verify labels
				labels := metric.GetLabel()
				assert.Equal(t, "method", labels[0].GetName())
				assert.Equal(t, "POST", labels[0].GetValue())
				assert.Equal(t, "path", labels[1].GetName())
				assert.Equal(t, "/tip-transfer", labels[1].GetValue())
				assert.Equal(t, "status", labels[2].GetName())
				assert.Equal(t, "200", labels[2].GetValue())
				// Verify value
				assert.Equal(t, float64(1), metric.GetCounter().GetValue())
			}
			if mf.GetName() == "micro_tip_http_requests_duration" {
				foundDuration = true
				metric := mf.GetMetric()[0]
				assert.Equal(t, uint64(1), metric.GetHistogram().GetSampleCount())
				assert.Equal(t, float64(0.150), metric.GetHistogram().GetSampleSum())
			}
		}

		assert.True(t, foundTotal)
		assert.True(t, foundDuration)
	})

	t.Run("Tip Metric", func(t *testing.T) {
		pm.IncTip(metrics.TipStatusSuccess)

		mfs, err := reg.Gather()
		assert.NoError(t, err)

		found := false
		for _, mf := range mfs {
			if mf.GetName() == "micro_tip_http_tips_total" {
				found = true
				metric := mf.GetMetric()[0]
				assert.Equal(t, "status", metric.GetLabel()[0].GetName())
				assert.Equal(t, "success", metric.GetLabel()[0].GetValue())
				assert.Equal(t, float64(1), metric.GetCounter().GetValue())
			}
		}
		assert.True(t, found)
	})

	t.Run("DB Transaction Duration Metric", func(t *testing.T) {
		pm.ObserveDBTransactionDuration("tip_transfer", 200*time.Millisecond)

		mfs, err := reg.Gather()
		assert.NoError(t, err)

		found := false
		for _, mf := range mfs {
			if mf.GetName() == "micro_tip_http_db_transaction_duration" {
				found = true
				metric := mf.GetMetric()[0]
				assert.Equal(t, uint64(1), metric.GetHistogram().GetSampleCount())
				assert.Equal(t, float64(0.200), metric.GetHistogram().GetSampleSum())
			}
		}
		assert.True(t, found)
	})

	t.Run("Cache Metrics", func(t *testing.T) {
		pm.IncCacheRead("balance", "hit")
		pm.ObserveCacheDuration("read_balance", 5*time.Millisecond)

		mfs, err := reg.Gather()
		assert.NoError(t, err)

		foundRead := false
		foundDuration := false

		for _, mf := range mfs {
			if mf.GetName() == "micro_tip_cache_cache_total" {
				foundRead = true
				metric := mf.GetMetric()[0]
				assert.Equal(t, "cache_name", metric.GetLabel()[0].GetName())
				assert.Equal(t, "balance", metric.GetLabel()[0].GetValue())
				assert.Equal(t, "status", metric.GetLabel()[1].GetName())
				assert.Equal(t, "hit", metric.GetLabel()[1].GetValue())
				assert.Equal(t, float64(1), metric.GetCounter().GetValue())
			}
			if mf.GetName() == "micro_tip_cache_cache_duration" {
				foundDuration = true
				metric := mf.GetMetric()[0]
				assert.Equal(t, uint64(1), metric.GetHistogram().GetSampleCount())
				assert.Equal(t, float64(0.005), metric.GetHistogram().GetSampleSum())
			}
		}

		assert.True(t, foundRead)
		assert.True(t, foundDuration)
	})
}
