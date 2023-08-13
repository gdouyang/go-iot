package logger_test

import (
	"go-iot/pkg/logger"
	"go-iot/pkg/option"
	"testing"
)

func TestLogger(t *testing.T) {
	logger.Init(&option.Options{})
	defer logger.Sync()
	// logger.InitNop()
	logger.Infof("test %s", "abc")
}
