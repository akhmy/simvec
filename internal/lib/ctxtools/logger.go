package ctxtools

import (
	"context"

	"go.uber.org/zap"
)

const loggerKey key = "logger"

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func Logger(ctx context.Context) (*zap.Logger, bool) {
	logger, ok := ctx.Value(loggerKey).(*zap.Logger)

	return logger, ok
}
