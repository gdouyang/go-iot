package logger

import "go.uber.org/zap"

type lazyLogBuilder struct {
	fn func() string
}

func (llb lazyLogBuilder) String() string {
	return llb.fn()
}

// Debugf is the wrapper of default logger Debugf.
func Debugf(template string, args ...interface{}) {
	defaultLogger.Debugf(template, args...)
}

// LazyDebug logs debug log in lazy mode. if debug log is disabled by configuration,
// it skips the the built of log message to improve performance
func LazyDebug(fn func() string) {
	defaultLogger.Debug(lazyLogBuilder{fn})
}

// Infof is the wrapper of default logger Infof.
func Infof(template string, args ...interface{}) {
	defaultLogger.Infof(template, args...)
}

// Warnf is the wrapper of default logger Warnf.
func Warnf(template string, args ...interface{}) {
	defaultLogger.Warnf(template, args...)
}

// Errorf is the wrapper of default logger Errorf.
func Errorf(template string, args ...interface{}) {
	defaultLogger.Errorf(template, args...)
}

// Sync syncs all logs, must be called after calling Init().
func Sync() {
	defaultLogger.Sync()
}

func IsDebug() bool {
	return lowestLevel == zap.DebugLevel
}
