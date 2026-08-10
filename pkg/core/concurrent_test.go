package core

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mock session：GetConInfo 返回共享 map（模拟各协议 session 的连接信息）
type mockSession struct {
	info map[string]any
}

func (s *mockSession) Disconnect() error  { return nil }
func (s *mockSession) GetDeviceId() string { return "d1" }
func (s *mockSession) SetDeviceId(string)  {}
func (s *mockSession) Close() error        { return nil }
func (s *mockSession) GetConInfo() map[string]any {
	return s.info
}

// 验证：PutSession（写 isDisconnect 标记）与 IsDeviceDisconnect（读 m[key]）并发，
// 修改前（标记存于 session 共享 map，读写无锁）触发 runtime 检测 fatal
func TestDeviceDisconnectConcurrentSafe(t *testing.T) {
	defaultSessions = NewMemorySessionManager()
	sess := &mockSession{info: map[string]any{}}

	var stop atomic.Bool
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stop.Load() {
			PutSession("d1", sess, false)
			DelSessionByUserDisconnect("d1")
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stop.Load() {
			_ = IsDeviceDisconnect("d1")
			DelSessionWithTimeoutCheck("d1")
		}
	}()
	time.Sleep(3 * time.Second)
	stop.Store(true)
	wg.Wait()
}
