package zaplogger

import (
	"fmt"

	"go.uber.org/zap"
)

func NewDev() (*zap.Logger, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("failed to create zap development: %w", err)
	}

	return logger, nil
}

func NewProd() (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create zap production: %w", err)
	}

	return logger, nil
}
