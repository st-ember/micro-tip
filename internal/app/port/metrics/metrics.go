package metrics

import "time"

type Metrics interface {
	// API Metrics
	IncHttpRequest(method, path, status string)
	ObserveHttpRequestDuration(method, path string, duration time.Duration)
	IncTip(status TipStatus)
	ObserveDBTransactionDuration(operation string, duration time.Duration)
	IncCacheRead(cacheName, status string)
	ObserveCacheDuration(operation string, duration time.Duration)
}

type TipStatus string

const (
	TipStatusStarted    TipStatus = "started"
	TipStatusSuccess    TipStatus = "success"
	TipStatusBalanceErr TipStatus = "balance_error"
	TipStatusDBErr      TipStatus = "db_error"
)
