package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func Init() error {
	// Write logs to both terminal and a file
	logFile, err := os.OpenFile("violations.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Terminal output config
	consoleConfig := zap.NewDevelopmentEncoderConfig()
	consoleConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(consoleConfig)

	// File output config (JSON)
	fileConfig := zap.NewProductionEncoderConfig()
	fileConfig.TimeKey = "timestamp"
	fileConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(fileConfig)

	// Combine both outputs
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
		zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), zapcore.DebugLevel),
	)

	Log = zap.New(core)
	return nil
}

func LogViolation(endpoint, method, direction, field, issue, expected, got string) {
	Log.Warn("Contract violation detected",
		zap.String("endpoint", endpoint),
		zap.String("method", method),
		zap.String("direction", direction),
		zap.String("field", field),
		zap.String("issue", issue),
		zap.String("expected", expected),
		zap.String("got", got),
		zap.String("timestamp", time.Now().Format(time.RFC3339)),
	)
}

func LogOK(endpoint, method, direction string) {
	Log.Info("Contract OK",
		zap.String("endpoint", endpoint),
		zap.String("method", method),
		zap.String("direction", direction),
		zap.String("timestamp", time.Now().Format(time.RFC3339)),
	)
}
