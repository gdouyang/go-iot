// 日志
package logger

import (
	"go-iot/pkg/option"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Init initializes logger.
func Init(opt *option.Options) {
	initDefault(opt, true)
}

// InitNop initializes all logger as nop, mainly for unit testing
func InitNop() {
	initDefault(&option.Options{Log: option.Log{Level: "debug"}}, false)
}

const (
	stdoutFilename = "logs/go-iot.log"
)

var (
	// 默认 Nop，避免未 Init 时测试/早期日志空指针；生产路径仍应调用 Init。
	// 使用 atomic 保证 Init/InitNop 与并发日志调用之间的可见性，避免全局初始化竞争。
	defaultLogger atomic.Pointer[zap.SugaredLogger]
	// atomicLevel 支持运行时动态调整日志级别（无需重建 logger）。
	atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	// lowestLevel 与 atomicLevel 同步，供 IsDebug 等快速判断。
	lowestLevel atomic.Int32
)

func init() {
	defaultLogger.Store(zap.NewNop().Sugar())
	lowestLevel.Store(int32(zap.InfoLevel))
}

func defaultEncoderConfig() zapcore.EncoderConfig {
	timeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(time.Now().Format("2006-01-02 15:04:05.000"))
	}

	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "", // no need
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "", // no need
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     timeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// ParseLevel 将配置字符串解析为 zap 级别；未知值按 info 处理。
func ParseLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zap.DebugLevel
	case "warn", "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "info", "":
		return zap.InfoLevel
	default:
		return zap.InfoLevel
	}
}

// SetLevel 动态设置全局日志级别（立即生效）。
func SetLevel(level string) {
	l := ParseLevel(level)
	atomicLevel.SetLevel(l)
	lowestLevel.Store(int32(l))
}

// GetLevel 返回当前日志级别字符串（debug/info/warn/error）。
func GetLevel() string {
	return atomicLevel.Level().String()
}

func initDefault(opt *option.Options, file bool) {
	encoderConfig := defaultEncoderConfig()
	SetLevel(opt.Log.Level)

	var goiotLF io.Writer = os.Stdout
	if file {
		var filename string = stdoutFilename
		if len(opt.Log.Dir) > 0 {
			filename = opt.Log.Dir
		}
		// os.Mkdir("logs", 0o777)
		goiotLF = &lumberjack.Logger{
			Filename:   filename, //filePath
			MaxSize:    100,      // 单个文件最大100M
			MaxBackups: 60,       // 多于 60 个日志文件后，清理较旧的日志
			MaxAge:     1,        // 一天一切割
			Compress:   false,    // disabled by default
		}
	}
	var format string = "text"
	if len(opt.Log.Format) > 0 {
		format = opt.Log.Format
	}

	opts := []zap.Option{zap.AddCaller(), zap.AddCallerSkip(1)}
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	// 使用 AtomicLevel，运行期 SetLevel 可立刻生效
	stdoutSyncer := zapcore.AddSync(os.Stdout)
	stdoutCore := zapcore.NewCore(encoder, stdoutSyncer, atomicLevel)

	goiotSyncer := zapcore.AddSync(goiotLF)
	goiotCore := zapcore.NewCore(encoder, goiotSyncer, atomicLevel)

	defaultCore := goiotCore
	if goiotLF != os.Stdout && goiotLF != os.Stderr {
		defaultCore = zapcore.NewTee(goiotCore, stdoutCore)
	}
	defaultLogger.Store(zap.New(defaultCore, opts...).Sugar())
}
