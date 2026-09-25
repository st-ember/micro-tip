package log

import (
	"context"
)

type Logger interface {
	InfoCtx(ctx context.Context, msg string, keysAndValues ...any)
	WarnCtx(ctx context.Context, msg string, keysAndValues ...any)
	ErrorCtx(ctx context.Context, msg string, err error, keysAndValues ...any)
	CriticalCtx(ctx context.Context, msg string, err error, keysAndValues ...any)
}
