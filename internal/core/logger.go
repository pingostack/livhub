package core

import "github.com/sirupsen/logrus"

var (
	defaultLogger = logrus.WithField("scope", "core")
)

func SetDefaultLogger(logger *logrus.Entry) {
	defaultLogger = logger
}

func DefaultLogger() *logrus.Entry {
	return defaultLogger
}
