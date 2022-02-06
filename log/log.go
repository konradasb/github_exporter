package log

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewZapLogger initializes a new *zap.Logger instance
func NewZapLogger(level string) (*zap.Logger, error) {
	if level == "off" {
		return zap.NewNop(), nil
	}

	cfg := zap.NewProductionConfig()

	var zapLevel zapcore.Level
	switch level {
	case "fatal":
		zapLevel = zap.FatalLevel
	case "panic":
		zapLevel = zap.PanicLevel
	case "error":
		zapLevel = zap.ErrorLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "debug":
		zapLevel = zap.DebugLevel
	}

	cfg.Level.SetLevel(zapLevel)

	logger, err := cfg.Build()
	if err != nil {
		return nil, errors.Wrap(err, "error building logger configuration")
	}

	return logger, nil
}
