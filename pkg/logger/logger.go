package logger

import (
	"fmt"

	"github.com/pion/logging"
	"github.com/sirupsen/logrus"
)

type Logger interface {
	Trace(args ...interface{})
	Tracef(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
}

var DefaultLogger Logger

func init() {
	DefaultLogger = &logrus.Logger{}
}

func SetDefaultLogger(logger Logger) {
	DefaultLogger = logger
}

type PionLogger struct {
	logger Logger
	scope  string
}

func (pl *PionLogger) Trace(msg string) {
	pl.logger.Trace(fmt.Sprintf("scope[%s]", pl.scope), msg)
}

func (pl *PionLogger) Tracef(format string, args ...interface{}) {
	pl.logger.Tracef(fmt.Sprintf("scope[%s] %s", pl.scope, format), args...)
}

func (pl *PionLogger) Debug(msg string) {
	pl.logger.Debug(fmt.Sprintf("scope[%s]", pl.scope), msg)
}

func (pl *PionLogger) Debugf(format string, args ...interface{}) {
	pl.logger.Debugf(fmt.Sprintf("scope[%s] %s", pl.scope, format), args...)
}

func (pl *PionLogger) Info(msg string) {
	pl.logger.Info(fmt.Sprintf("scope[%s]", pl.scope), msg)
}

func (pl *PionLogger) Infof(format string, args ...interface{}) {
	pl.logger.Infof(fmt.Sprintf("scope[%s] %s", pl.scope, format), args...)
}

func (pl *PionLogger) Warn(msg string) {
	pl.logger.Warn(fmt.Sprintf("scope[%s]", pl.scope), msg)
}

func (pl *PionLogger) Warnf(format string, args ...interface{}) {
	pl.logger.Warnf(fmt.Sprintf("scope[%s] %s", pl.scope, format), args...)
}

func (pl *PionLogger) Error(msg string) {
	pl.logger.Error(fmt.Sprintf("scope[%s]", pl.scope), msg)
}

func (pl *PionLogger) Errorf(format string, args ...interface{}) {
	pl.logger.Errorf(fmt.Sprintf("scope[%s] %s", pl.scope, format), args...)
}

type PionLoggerFactory struct {
	defaultLogger Logger
}

func (lf *PionLoggerFactory) NewLogger(scope string) logging.LeveledLogger {
	return &PionLogger{
		logger: lf.defaultLogger,
		scope:  scope,
	}
}

func NewPionLoggerFactory(l Logger) *PionLoggerFactory {
	if l == nil {
		l = DefaultLogger
	}

	return &PionLoggerFactory{
		defaultLogger: l,
	}
}

func Trace(args ...interface{}) {
	DefaultLogger.Trace(insertSpace(args))
}

func Tracef(format string, args ...interface{}) {
	DefaultLogger.Tracef(format, args...)
}

func insertSpace(args ...interface{}) []interface{} {
	newArgs := make([]interface{}, 0, len(args)*2)
	for _, arg := range args {
		newArgs = append(newArgs, arg, " ")
	}
	return newArgs
}

func Debug(args ...interface{}) {
	DefaultLogger.Debug(insertSpace(args)...)
}

func Debugf(format string, args ...interface{}) {
	DefaultLogger.Debugf(format, args...)
}

func Info(args ...interface{}) {
	DefaultLogger.Info(insertSpace(args)...)
}

func Infof(format string, args ...interface{}) {
	DefaultLogger.Infof(format, args...)
}

func Warn(args ...interface{}) {
	DefaultLogger.Warn(insertSpace(args)...)
}

func Warnf(format string, args ...interface{}) {
	DefaultLogger.Warnf(format, args...)
}

func Error(args ...interface{}) {
	DefaultLogger.Error(insertSpace(args)...)
}

func Errorf(format string, args ...interface{}) {
	DefaultLogger.Errorf(format, args...)
}
