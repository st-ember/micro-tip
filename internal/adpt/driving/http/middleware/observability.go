package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/st-ember/microtip/internal/app/port/log"
	"github.com/st-ember/microtip/internal/app/port/metrics"
)

func Observability(m metrics.Metrics, logger log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 1. Trace ID propagation
		traceID := c.GetHeader("X-Request-ID")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		c.Set("trace_id", traceID)
		c.Header("X-Request-ID", traceID)

		ctx := context.WithValue(c.Request.Context(), log.TraceIDKey, traceID)
		c.Request = c.Request.WithContext(ctx)

		// 2. Process request
		c.Next()

		// 3. Gather data
		duration := time.Since(start)
		status := c.Writer.Status()
		statusStr := strconv.Itoa(status)
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = "unknown_route"
		}

		// 4. Record Metrics
		m.IncHttpRequest(method, path, statusStr)
		m.ObserveHttpRequestDuration(method, path, duration)

		// 5. Record Structured Log
		var reqErr error
		if len(c.Errors) > 0 {
			reqErr = c.Errors.Last().Err
		}

		logAttrs := []any{
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", duration.Milliseconds(),
			"trace_id", traceID,
		}

		if status >= 500 {
			logger.ErrorCtx(c.Request.Context(), "http request failed", reqErr, logAttrs...)
		} else if status >= 400 {
			logger.WarnCtx(c.Request.Context(), "http request client error", logAttrs...)
		} else {
			logger.InfoCtx(c.Request.Context(), "http request completed", logAttrs...)
		}
	}
}
