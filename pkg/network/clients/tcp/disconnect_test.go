package tcpclient

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 验证：TcpSession.Disconnect() 幂等。API Close 与 readLoop defer Disconnect()
// 可能并发触发；修复前 isClose 为普通 bool 无锁读写，并发下会 double close(s.done)
// panic: close of closed channel。
func TestTcpSession_DisconnectIdempotentAndConcurrent(t *testing.T) {
	ln, _ := net.Pipe()
	defer ln.Close()

	s := &TcpSession{
		deviceId: "tci-0",
		conn:     ln,
		done:     make(chan struct{}),
		info:     map[string]any{},
	}

	// 并发调用 Disconnect 多次不得 panic、不得卡住
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Disconnect()
		}()
	}
	wg.Wait()

	// 关闭后状态已置位
	<-s.done

	// 关闭后再次调用幂等，无 panic
	require.NoError(t, s.Disconnect())
}

// 验证：TcpSession.GetConInfo() 并发调用（旧实现每次写共享 s.info map，
// API 多请求并发查询连接信息会触发 fatal: concurrent map read and map write）。
// 修复：GetConInfo 持 s.mu 写 s.info。
func TestTcpClientSession_GetConInfoConcurrentSafe(t *testing.T) {
	ln, _ := net.Pipe()
	defer ln.Close()

	s := &TcpSession{
		deviceId: "tci-1",
		conn:     ln,
		done:     make(chan struct{}),
		info:     map[string]any{},
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
