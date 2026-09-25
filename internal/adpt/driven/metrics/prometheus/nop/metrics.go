package nopmetrics

import (
	"time"

	"github.com/st-ember/microtip/internal/app/port/metrics"
)

type NopMetrics struct{}

func NewNopMetrics() *NopMetrics {
	return &NopMetrics{}
}

func (*NopMetrics) IncHttpRequest(method, path, status string) {}

func (*NopMetrics) ObserveHttpRequestDuration(method, path string, duration time.Duration) {}

func (*NopMetrics) IncTip(status metrics.TipStatus) {}

func (*NopMetrics) ObserveDBTransactionDuration(operation string, duration time.Duration) {}

func (*NopMetrics) IncCacheRead(cacheName, status string) {}

func (*NopMetrics) ObserveCacheDuration(operation string, duration time.Duration) {}
