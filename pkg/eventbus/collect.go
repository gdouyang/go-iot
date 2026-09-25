// Package eventbus 提供系统内部事件总线（发布/订阅）能力。
// collect.go 实现了短程临时消息收集器（Collector），用于在指定时间窗口或指定数量内捕获特定 Topic 的事件流。
//
// 核心机制与特点：
// 1. 临时订阅：按需创建、用后即退（UnSubscribe），避免常驻订阅导致的内存或通道泄漏。
// 2. 限时与限量：支持设置超时时间（wait）和消息上限（limit），先到先止。
// 3. 非阻塞缓冲：回调写入 channel 采用 select-default，若缓冲区满则丢弃超量消息，避免阻塞全局 EventBus。
// 4. 安全排空（Drain）：退订后保留短暂的排空窗口（30ms），确保并发异步分发的回调消息被全部捕获。
//
// 典型应用场景：
// - 编解码演练试跑（pkg/codec/dryrun.go）：临时监听某次执行中 console.log 产生的调试日志；
// - 管理端 AI 智能助手（pkg/agent/tools_debug.go）：临时抓取真实设备当前上报的控制台日志进行故障排查。
package eventbus

import (
	"strings"
	"sync"
	"time"
)

const (
	collectDefaultWait = 3 * time.Second        // 默认等待收集时长
	collectMaxWait     = 15 * time.Second       // 最大等待收集时长（防止请求长时间挂起）
	collectMinWait     = 200 * time.Millisecond // 最小等待收集时长
	collectDefaultLim  = 50                     // 默认收集消息上限条数
	collectMaxLim      = 200                    // 最大收集消息上限条数
	collectDrain       = 30 * time.Millisecond  // 退订后等待异步 Publish 回调落地的排空时间
)

// Collector 临时订阅收集器，持续监听指定的 Topic 模式直到调用 Stop
type Collector struct {
	pattern string          // 订阅的主题模式（支持通配符）
	handler func(Message)  // 注册到 EventBus 的消息处理回调
	ch      chan Message    // 消息缓冲通道
	mu      sync.Mutex      // 保护 stopped 状态
	stopped bool            // 是否已停止并退订（保证 Stop 幂等）
}

// Listen 开始针对指定 pattern 的临时订阅，返回 Collector 收集器。
// 调用方必须在完成收集后显式调用 Stop() 以退订并释放资源。
// buf 为缓冲通道大小，若 <= 0 则默认为 64。
func Listen(pattern string, buf int) *Collector {
	if buf < 1 {
		buf = 64
	}
	c := &Collector{pattern: pattern, ch: make(chan Message, buf)}
	c.handler = func(msg Message) {
		select {
		case c.ch <- msg:
		default:
			// 缓冲满时非阻塞丢弃，避免拖垮全局 EventBus 发布方
		}
	}
	Subscribe(pattern, c.handler)
	return c
}

// Stop 停止订阅、取消 EventBus 注册、排空并返回缓冲区中已接收到的全部消息（幂等）
func (c *Collector) Stop() []Message {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return nil
	}
	c.stopped = true
	c.mu.Unlock()

	UnSubscribe(c.pattern, c.handler)
	time.Sleep(collectDrain) // 等待正在并发异步执行的 publish 回调完成投递
	var out []Message
	for {
		select {
		case m := <-c.ch:
			out = append(out, m)
		default:
			return out
		}
	}
}

// Collect 在 pattern 主题上进行阻塞式收集，最多等待 wait 时长或收集到 limit 条消息后自动停止并退订。
func Collect(pattern string, limit int, wait time.Duration) []Message {
	limit = clampLimit(limit)
	wait = clampWait(wait)
	c := Listen(pattern, limit)
	timer := time.NewTimer(wait)
	defer timer.Stop()
	var out []Message
	for {
		select {
		case m := <-c.ch:
			out = append(out, m)
			if len(out) >= limit {
				_ = c.Stop()
				return out[:limit]
			}
		case <-timer.C:
			rest := c.Stop()
			out = append(out, rest...)
			if len(out) > limit {
				out = out[:limit]
			}
			return out
		}
	}
}

// CollectDebug 专用于收集设备或产品的调试日志（/device/{productId}/{deviceId}/debug）。
// 若 deviceId 为空，则监听该产品下的全部设备（*）。
// 返回强类型的 DebugMessage 列表。
func CollectDebug(productId, deviceId string, limit int, wait time.Duration) []*DebugMessage {
	productId = strings.TrimSpace(productId)
	if productId == "" {
		return nil
	}
	dev := strings.TrimSpace(deviceId)
	if dev == "" {
		dev = "*"
	}
	raw := Collect(GetDebugTopic(productId, dev), limit, wait)
	out := make([]*DebugMessage, 0, len(raw))
	for _, m := range raw {
		if d, ok := m.(*DebugMessage); ok {
			out = append(out, d)
		}
	}
	return out
}

// clampLimit 对收集数量上限进行边界钳位约束 [1, collectMaxLim]
func clampLimit(n int) int {
	if n <= 0 {
		return collectDefaultLim
	}
	if n > collectMaxLim {
		return collectMaxLim
	}
	return n
}

// clampWait 对收集超时时间进行边界钳位约束 [collectMinWait, collectMaxWait]
func clampWait(d time.Duration) time.Duration {
	if d <= 0 {
		return collectDefaultWait
	}
	if d < collectMinWait {
		return collectMinWait
	}
	if d > collectMaxWait {
		return collectMaxWait
	}
	return d
}
