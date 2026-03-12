package logs

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Initialize the logger with custom configuration
func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.StacktraceKey = ""

	var err error
	log, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
}

// Info logs an informational message with optional fields.
func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

// Warn logs a warning message with optional fields.
func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
}

// Debug logs a debug message with optional fields.
func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

// Error logs an error message. It accepts either a string or an error type as the message.
func Error(msg interface{}, fields ...zap.Field) {
	switch v := msg.(type) {
	case error:
		log.Error(v.Error(), fields...)
	case string:
		log.Error(v, fields...)
	}
}
