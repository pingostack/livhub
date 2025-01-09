package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// zapLogger implements the Logger interface using zap
type zapLogger struct {
	logger *zap.Logger
}

// NewLogger creates a new zap logger with lumberjack for log rotation
func NewLogger() Logger {
	return NewLoggerWithConfig(zap.NewDevelopmentConfig())
}

// NewLoggerWithConfig creates a new zap logger with the provided configuration
func NewLoggerWithConfig(config zap.Config) Logger {
	// Configure lumberjack for log rotation
	lumberjackLogger := &lumberjack.Logger{
		Filename:   "logs/app.log", // Log file path
		MaxSize:    10,             // Maximum size of each log file (MB)
		MaxBackups: 5,              // Maximum number of backup files to keep
		MaxAge:     30,             // Maximum number of days to keep log files
		Compress:   true,           // Whether to compress backup log files
	}

	// Set the encoder for the config
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	// Create a zapcore.Core that writes to lumberjack
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(config.EncoderConfig),
		zapcore.AddSync(lumberjackLogger), // Use lumberjack as output
		config.Level,                      // Log level
	)

	// Create the logger
	logger := zap.New(core)

	return &zapLogger{logger: logger}
}

// WithField adds a field to the logger
func (l *zapLogger) WithField(key string, value interface{}) Logger {
	return &zapLogger{logger: l.logger.With(zap.Any(key, value))}
}

// WithFields adds multiple fields to the logger
func (l *zapLogger) WithFields(fields map[string]interface{}) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &zapLogger{logger: l.logger.With(zapFields...)}
}

// WithError adds an error field to the logger
func (l *zapLogger) WithError(err error) Logger {
	return l.WithField("error", err)
}

// Debug logs a message at DebugLevel
func (l *zapLogger) Debug(args ...interface{}) {
	l.logger.Debug(fmt.Sprint(args...))
}

// Debugf logs a formatted message at DebugLevel
func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

// Info logs a message at InfoLevel
func (l *zapLogger) Info(args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...))
}

// Infof logs a formatted message at InfoLevel
func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

// Warn logs a message at WarnLevel
func (l *zapLogger) Warn(args ...interface{}) {
	l.logger.Warn(fmt.Sprint(args...))
}

// Warnf logs a formatted message at WarnLevel
func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

// Error logs a message at ErrorLevel
func (l *zapLogger) Error(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
}

// Errorf logs a formatted message at ErrorLevel
func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

// Fatal logs a message at FatalLevel
func (l *zapLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(fmt.Sprint(args...))
}

// Fatalf logs a formatted message at FatalLevel
func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal(fmt.Sprintf(format, args...))
}

// Panic logs a message at PanicLevel
func (l *zapLogger) Panic(args ...interface{}) {
	l.logger.Panic(fmt.Sprint(args...))
}

// Panicf logs a formatted message at PanicLevel
func (l *zapLogger) Panicf(format string, args ...interface{}) {
	l.logger.Panic(fmt.Sprintf(format, args...))
}

// SetLevel sets the log level of the logger
func (l *zapLogger) SetLevel(level Level) {
	var zapLevel zapcore.Level
	switch level {
	case DebugLevel:
		zapLevel = zapcore.DebugLevel
	case InfoLevel:
		zapLevel = zapcore.InfoLevel
	case WarnLevel:
		zapLevel = zapcore.WarnLevel
	case ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	case FatalLevel:
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	l.logger = l.logger.WithOptions(zap.IncreaseLevel(zapLevel))
}

func init() {
	defaultLogger := NewLogger()
	SetDefaultLogger(defaultLogger)
}
