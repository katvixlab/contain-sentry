package config

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	once   sync.Once
)

type loggerKey struct{}

func WithLogger(ctx context.Context, l *zap.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey{}, l)
}

func FromCtx(ctx context.Context) *zap.Logger {
	once.Do(func() {
		logger = NewLogger(WithConfig(NewDefaultConfig()))
	})

	if ctx == nil {
		return logger
	}

	if l, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return logger
}
