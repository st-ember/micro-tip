package slogger

import (
	"context"
	"log/slog"

	"github.com/st-ember/microtip/internal/app/port/log"
)

type Slogger struct {
	logger *slog.Logger
}

func NewSlogger(logger *slog.Logger) log.Logger {
	return &Slogger{logger}
}

func (l *Slogger) InfoCtx(ctx context.Context, msg string, keysAndValues ...any) {
	args := l.injectTraceID(ctx, keysAndValues)
	l.logger.InfoContext(ctx, msg, args...)
}

func (l *Slogger) WarnCtx(ctx context.Context, msg string, keysAndValues ...any) {
	args := l.injectTraceID(ctx, keysAndValues)
	l.logger.WarnContext(ctx, msg, args...)
}

func (l *Slogger) ErrorCtx(ctx context.Context, msg string, err error, keysAndValues ...any) {
	args := append([]any{"error", err}, keysAndValues...)
	args = l.injectTraceID(ctx, args)
	l.logger.ErrorContext(ctx, msg, args...)
}

func (l *Slogger) CriticalCtx(ctx context.Context, msg string, err error, keysAndValues ...any) {
	args := append([]any{"error", err}, keysAndValues...)
	args = l.injectTraceID(ctx, args)
	// TODO: add alarm logic
	l.logger.ErrorContext(ctx, msg, args...)
}

// Helper to automatically inject trace_id from context if present
func (l *Slogger) injectTraceID(ctx context.Context, args []any) []any {
	if ctx == nil {
		return args
	}
	if tid, ok := ctx.Value(log.TraceIDKey).(string); ok && tid != "" {
		// Prepend or append trace_id to the key-value pairs
		return append(args, "trace_id", tid)
	}
	return args
}
