package prometheusmetrics

import (
	"database/sql"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/st-ember/microtip/internal/app/port/metrics"
)

type PrometheusMetrics struct {
	httpRequestTotal    *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	tipsTotal           *prometheus.CounterVec
	dbTxDuration        *prometheus.HistogramVec
	cacheReadTotal      *prometheus.CounterVec
	cacheDuration       *prometheus.HistogramVec
}

func NewPrometheusMetrics(reg *prometheus.Registry, db *sql.DB) metrics.Metrics {
	// Register standard DB stats to custom registry
	dbStatsCollector := collectors.NewDBStatsCollector(db, "microtip")
	reg.MustRegister(dbStatsCollector)

	// Create a factory tied to custom registry
	factory := promauto.With(reg)

	return &PrometheusMetrics{
		// 3. Use factory instead of promauto directly
		httpRequestTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "micro_tip",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed.",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "micro_tip",
				Subsystem: "http",
				Name:      "requests_duration",
				Help:      "Duration of HTTP requests.",
			},
			[]string{"method", "path"},
		),
		tipsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "micro_tip",
				Subsystem: "http",
				Name:      "tips_total",
				Help:      "Total number of tips processed.",
			},
			[]string{"status"},
		),
		dbTxDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "micro_tip",
				Subsystem: "http",
				Name:      "db_transaction_duration",
				Help:      "Duration of db tip transactions.",
			},
			[]string{"operation"},
		),
		cacheReadTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "micro_tip",
				Subsystem: "cache",
				Name:      "cache_total",
				Help:      "Total number of cache read operations.",
			},
			[]string{"cache_name", "status"},
		),
		cacheDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "micro_tip",
				Subsystem: "cache",
				Name:      "cache_duration",
				Help:      "Duration of cache operations.",
			},
			[]string{"operation"},
		),
	}
}

// API Metrics
func (pm *PrometheusMetrics) IncHttpRequest(method, path, status string) {
	pm.httpRequestTotal.WithLabelValues(method, path, status).Inc()
}

func (pm *PrometheusMetrics) ObserveHttpRequestDuration(method, path string, duration time.Duration) {
	pm.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) IncTip(status metrics.TipStatus) {
	pm.tipsTotal.WithLabelValues(string(status)).Inc()
}

func (pm *PrometheusMetrics) ObserveDBTransactionDuration(operation string, duration time.Duration) {
	pm.dbTxDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) IncCacheRead(cacheName, status string) {
	pm.cacheReadTotal.WithLabelValues(cacheName, status).Inc()
}

func (pm *PrometheusMetrics) ObserveCacheDuration(operation string, duration time.Duration) {
	pm.cacheDuration.WithLabelValues(operation).Observe(duration.Seconds())
}
