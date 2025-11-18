package logger

import (
	"context"

	"audio-scraper/internal/ports"
)

type ctxKey struct{}

func Into(ctx context.Context, l ports.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func From(ctx context.Context) ports.Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		if lg, ok := v.(ports.Logger); ok {
			return lg
		}
	}
	return NewLogger()
}
