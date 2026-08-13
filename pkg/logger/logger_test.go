package logger_test

import (
	"go-iot/pkg/logger"
	"go-iot/pkg/option"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLogger(t *testing.T) {
	logger.Init(&option.Options{})
	defer logger.Sync()
	// logger.InitNop()
	logger.Infof("test %s", "abc")
}

func TestSetLevelDynamic(t *testing.T) {
	logger.Init(&option.Options{Log: option.Log{Level: "info"}})
	defer logger.Sync()

	require.Equal(t, "info", logger.GetLevel())
	require.False(t, logger.IsDebug())

	logger.SetLevel("debug")
	require.Equal(t, "debug", logger.GetLevel())
	require.True(t, logger.IsDebug())

	logger.SetLevel("WARN")
	require.Equal(t, "warn", logger.GetLevel())
	require.False(t, logger.IsDebug())

	logger.SetLevel("error")
	require.Equal(t, "error", logger.GetLevel())
}

func TestParseLevel(t *testing.T) {
	require.Equal(t, zap.DebugLevel, logger.ParseLevel("debug"))
	require.Equal(t, zap.InfoLevel, logger.ParseLevel("info"))
	require.Equal(t, zap.InfoLevel, logger.ParseLevel(""))
	require.Equal(t, zap.InfoLevel, logger.ParseLevel("unknown"))
	require.Equal(t, zap.WarnLevel, logger.ParseLevel("warning"))
	require.Equal(t, zap.ErrorLevel, logger.ParseLevel("ERROR"))
}
