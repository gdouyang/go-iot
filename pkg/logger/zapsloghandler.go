package logger

import (
	"context"
	"log/slog"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SugaredHandler 实现slog.Handler接口
type SugaredHandler struct {
	slog.Logger
	logger *zap.SugaredLogger
	groups []string // 当前组名栈
}

// NewSugaredHandler 创建适配器
func NewSugaredHandler() *SugaredHandler {
	return &SugaredHandler{
		logger: defaultLogger,
	}
}

// Enabled 判断是否启用某级别
func (h *SugaredHandler) Enabled(_ context.Context, level slog.Level) bool {
	zapLevel := toZapLevel(level)
	return h.logger.Desugar().Core().Enabled(zapLevel)
}

// Handle 处理日志记录
func (h *SugaredHandler) Handle(_ context.Context, r slog.Record) error {
	args := h.convertArgs(r)

	switch r.Level {
	case slog.LevelDebug:
		h.logger.Debugw(r.Message, args...)
	case slog.LevelInfo:
		h.logger.Infow(r.Message, args...)
	case slog.LevelWarn:
		h.logger.Warnw(r.Message, args...)
	case slog.LevelError:
		h.logger.Errorw(r.Message, args...)
	}
	return nil
}

// WithAttrs 添加固定属性
func (h *SugaredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return &SugaredHandler{
		logger: h.logger.With(h.convertArgs1(attrs)...),
		groups: h.groups,
	}
}

// WithGroup 添加日志组
func (h *SugaredHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &SugaredHandler{
		logger: h.logger,
		groups: append(h.groups, name),
	}
}

// 转换slog级别到Zap级别
func toZapLevel(level slog.Level) zapcore.Level {
	switch {
	case level < slog.LevelInfo:
		return zapcore.DebugLevel
	case level < slog.LevelWarn:
		return zapcore.InfoLevel
	case level < slog.LevelError:
		return zapcore.WarnLevel
	default:
		return zapcore.ErrorLevel
	}
}

// 递归转换属性和组
func (h *SugaredHandler) convertArgs(r slog.Record) []interface{} {
	args := make([]interface{}, 0, r.NumAttrs()*2)
	r.Attrs(func(a slog.Attr) bool {
		args = append(args, h.processAttr(a)...)
		return true
	})
	return args
}

// 递归转换属性和组
func (h *SugaredHandler) convertArgs1(attrs []slog.Attr) []interface{} {
	args := make([]interface{}, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, h.processAttr(attr)...)
	}
	return args
}

// 处理单个属性（含组嵌套）
func (h *SugaredHandler) processAttr(attr slog.Attr) []interface{} {
	// 处理LogValuer
	attr.Value = attr.Value.Resolve()

	// 生成完整键名
	fullKey := h.buildKey(attr.Key)

	switch attr.Value.Kind() {
	case slog.KindGroup:
		// 递归处理组属性
		subHandler := &SugaredHandler{
			groups: append(h.groups, attr.Key),
		}
		return subHandler.convertArgs1(attr.Value.Group())
	default:
		return []interface{}{fullKey, h.convertValue(attr.Value)}
	}
}

// 构建完整键名（含组前缀）
func (h *SugaredHandler) buildKey(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	return strings.Join(append(h.groups, key), ".")
}

// 转换属性值
func (h *SugaredHandler) convertValue(v slog.Value) interface{} {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindTime:
		return v.Time()
	case slog.KindAny:
		if err, ok := v.Any().(error); ok {
			return err
		}
		return v.Any()
	default:
		return v.Any()
	}
}
