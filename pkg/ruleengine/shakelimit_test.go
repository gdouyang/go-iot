package ruleengine

import (
	"testing"
	"time"
)

// 重复 Stop（close）必须幂等：init goroutine 退出后再次 close 不得阻塞
func TestShakeLimitCloseIdempotent(t *testing.T) {
	sl := &ShakeLimit{Enabled: true, Time: 1, Threshold: 1}
	sl.init(func(deviceId string, data map[string]any) {})
	sl.close() // 第一次：正常退出
	time.Sleep(100 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		sl.close() // 第二次：修复前无接收者，永久阻塞
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("重复 close 阻塞")
	}
}
