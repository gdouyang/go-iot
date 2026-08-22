package mqtt5

import (
	"context"
	"log/slog"
	"strings"

	logs "go-iot/pkg/logger"
)

// mqttLogHandler 包装底层日志 Handler，专门过滤/降级 Mochi MQTT 的良性断连日志
type mqttLogHandler struct {
	slog.Handler
}

func newMqttLogger() *slog.Logger {
	baseHandler := logs.NewSugaredHandler()
	return slog.New(&mqttLogHandler{Handler: baseHandler})
}

func (h *mqttLogHandler) Handle(ctx context.Context, r slog.Record) error {
	// 如果是 WARN 级别的良性网络断开（EOF、i/o timeout等），将其降级为 DEBUG 级别
	if r.Level == slog.LevelWarn && isBenignMqttError(r) {
		newRecord := slog.NewRecord(r.Time, slog.LevelDebug, r.Message, r.PC)
		r.Attrs(func(a slog.Attr) bool {
			newRecord.AddAttrs(a)
			return true
		})
		return h.Handler.Handle(ctx, newRecord)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *mqttLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &mqttLogHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *mqttLogHandler) WithGroup(name string) slog.Handler {
	return &mqttLogHandler{Handler: h.Handler.WithGroup(name)}
}

// 检查是否为良性断连错误
func isBenignMqttError(r slog.Record) bool {
	var isBenign bool
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "error" {
			errStr := a.Value.String()
			if isBenignErrorStr(errStr) {
				isBenign = true
				return false
			}
		}
		return true
	})
	if isBenign {
		return true
	}
	return isBenignErrorStr(r.Message)
}

func isBenignErrorStr(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s == "EOF" ||
		strings.Contains(s, "EOF") ||
		strings.Contains(s, "use of closed network connection") ||
		strings.Contains(s, "i/o timeout") ||
		strings.Contains(s, "connection reset by peer") ||
		strings.Contains(s, "broken pipe")
}
