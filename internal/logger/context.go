package logger

import (
	"context"
)

type ctxKey struct{}

func Into(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func From(ctx context.Context) Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		if lg, ok := v.(Logger); ok {
			return lg
		}
	}
	return NewLogger()
}
