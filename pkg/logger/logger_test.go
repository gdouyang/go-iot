package logger_test

import (
	"go-iot/pkg/logger"
	"testing"
)

func TestLogger(t *testing.T) {
	logger.Init(func(key string, call func(string)) {})
	defer logger.Sync()
	// logger.InitNop()
	logger.Infof("test %s", "abc")
}
