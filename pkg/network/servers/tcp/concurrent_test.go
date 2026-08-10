package tcpserver

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 验证：TcpSession.GetConInfo() 并发调用（旧实现每次写共享 s.info map，在 API
// 多请求并发查询连接信息时会触发 fatal: concurrent map read and map write）。
// 修复：GetConInfo 持 s.Lock 写 s.info。
func TestTcpSession_GetConInfoConcurrentSafe(t *testing.T) {
	ln, _ := net.Pipe()
	defer ln.Close()

	s := &TcpSession{
		id:         "tcp-test",
		conn:       ln,
		statusFlag: Connected,
		info:       map[string]any{},
	}

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
	time.Sleep(1 * time.Second)
	stop.Store(true)
	wg.Wait()
}
