package modbus

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// 验证：ModbusSession.Disconnect() 幂等。API Close 与 reload/codec 关闭路径
// 可能并发触发；修复前 stopped 为普通 bool 无锁读写后 close(s.done) 会发生
// close of closed channel panic。
func TestModbusSession_DisconnectIdempotentAndConcurrent(t *testing.T) {
	s := newSession()

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

	// done 已关闭
	<-s.done

	// 关闭后再次调用幂等，无 panic；stopped 已置位
	require.NoError(t, s.Disconnect())
	require.True(t, s.stopped.Load(), "stopped should be set")
}

// 验证：Disconnect 后 lockAddress 不再阻塞/不再 send on closed channel panic。
// 修复前 Disconnect 会 close(s.lock)，并发 lockAddress 的 s.lock<-true 会 panic。
func TestModbusSession_LockAddressAfterDisconnectSafe(t *testing.T) {
	s := newSession()
	s.tcpInfo = &TcpInfo{Address: "127.0.0.1", Port: 502}
	_ = s.Disconnect()

	// 随后调用 lockAddress 必须快速返回错误而非阻塞或 panic
	require.Error(t, s.lockAddress(s.lockableAddress(nil)))
}
