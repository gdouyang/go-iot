package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type lazyLogBuilder struct {
	fn func() string
}

func (llb lazyLogBuilder) String() string {
	return llb.fn()
}

// Debugf is the wrapper of default logger Debugf.
func Debugf(template string, args ...interface{}) {
	defaultLogger.Load().Debugf(template, args...)
}

// LazyDebug logs debug log in lazy mode. if debug log is disabled by configuration,
// it skips the the built of log message to improve performance
func LazyDebug(fn func() string) {
	defaultLogger.Load().Debug(lazyLogBuilder{fn})
}

// Infof is the wrapper of default logger Infof.
func Infof(template string, args ...interface{}) {
	defaultLogger.Load().Infof(template, args...)
}

// Warnf is the wrapper of default logger Warnf.
func Warnf(template string, args ...interface{}) {
	defaultLogger.Load().Warnf(template, args...)
}

// Errorf is the wrapper of default logger Errorf.
func Errorf(template string, args ...interface{}) {
	defaultLogger.Load().Errorf(template, args...)
}

// Sync syncs all logs, must be called after calling Init().
func Sync() {
	if l := defaultLogger.Load(); l != nil {
		_ = l.Sync()
	}
}

func IsDebug() bool {
	return zapcore.Level(lowestLevel.Load()) == zap.DebugLevel
}
