package common

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetupLogger is used to create a new human friendly
// logger using Uber's zap package. This will change
// the encode level to include color for debugging.
func SetupLogger() (logger *zap.Logger) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ = config.Build()
	zap.ReplaceGlobals(logger)
	return
}
