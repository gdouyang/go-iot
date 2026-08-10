package websocketserver

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 验证：GetConInfo() 并发调用（旧实现每次写共享 s.info map，写路径触发
// runtime 检测）会 fatal: concurrent map read and map write
func TestGetConInfoConcurrentSafe(t *testing.T) {
	s := &WebsocketSession{}

	var stop atomic.Bool
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				_ = s.GetConInfo()
			}
		}()
	}
	time.Sleep(3 * time.Second)
	stop.Store(true)
	wg.Wait()
}
