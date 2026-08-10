// 日志
package logger

import (
	"go-iot/pkg/option"
	"io"
	"os"
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
	lowestLevel   atomic.Int32
)

func init() {
	defaultLogger.Store(zap.NewNop().Sugar())
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

func initDefault(opt *option.Options, file bool) {
	encoderConfig := defaultEncoderConfig()

	switch opt.Log.Level {
	case "debug":
		lowestLevel.Store(int32(zap.DebugLevel))
	case "warn":
		lowestLevel.Store(int32(zap.WarnLevel))
	case "error":
		lowestLevel.Store(int32(zap.ErrorLevel))
	}

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
	level := zapcore.Level(lowestLevel.Load())
	stdoutSyncer := zapcore.AddSync(os.Stdout)
	stdoutCore := zapcore.NewCore(encoder, stdoutSyncer, level)

	goiotSyncer := zapcore.AddSync(goiotLF)
	goiotCore := zapcore.NewCore(encoder, goiotSyncer, level)

	defaultCore := goiotCore
	if goiotLF != os.Stdout && goiotLF != os.Stderr {
		defaultCore = zapcore.NewTee(goiotCore, stdoutCore)
	}
	defaultLogger.Store(zap.New(defaultCore, opts...).Sugar())
}
