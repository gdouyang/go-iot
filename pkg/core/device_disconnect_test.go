package core

import (
	"sync"
	"testing"
)

// 最小 DeviceStore 桩：仅缓存设备，避免导入 pkg/store 形成 import cycle
type mockDeviceStore struct {
	m  sync.Map
	p  *Product
}

func (s *mockDeviceStore) Id() string                        { return "test" }
func (s *mockDeviceStore) GetDevice(id string) *Device       { v, ok := s.m.Load(id); if ok { return v.(*Device) }; return nil }
func (s *mockDeviceStore) PutDevice(d *Device) error         { s.m.Store(d.Id, d); return nil }
func (s *mockDeviceStore) DelDevice(id string)               { s.m.Delete(id) }
func (s *mockDeviceStore) RefreshOfflineTimeout(id string)   {}
func (s *mockDeviceStore) GetDeviceData(id, k string) string { return "" }
func (s *mockDeviceStore) SetDeviceData(id, k string, v any) {}
func (s *mockDeviceStore) GetProduct(id string) *Product     { return s.p }
func (s *mockDeviceStore) PutProduct(p *Product) error       { s.p = p; return nil }
func (s *mockDeviceStore) DelProduct(id string)              { s.p = nil }

// deviceDisconnect 标记的生命周期：随 session 增删，缺失等价于 false（在线）
func TestDeviceDisconnectLifecycle(t *testing.T) {
	defaultSessions = NewMemorySessionManager()
	defaultStore = &mockDeviceStore{}

	dev := &Device{Id: "d1", ProductId: "p1", Config: map[string]string{}}
	if err := defaultStore.PutDevice(dev); err != nil {
		t.Fatal(err)
	}
	sess := &mockSession{info: map[string]any{}}

	// 1. 未上线：无 session → 视为离线
	if !IsDeviceDisconnect("d1") {
		t.Error("未上线设备应视为离线")
	}

	// 2. 上线 → 在线
	PutSession("d1", sess, false)
	if IsDeviceDisconnect("d1") {
		t.Error("上线后应在线")
	}
	if _, ok := deviceDisconnect.Load("d1"); !ok {
		t.Error("上线后应有断开标记")
	}

	// 3. 正常下线 → 离线，key 随 session 清除
	DelSessionByUserDisconnect("d1")
	if !IsDeviceDisconnect("d1") {
		t.Error("下线后应离线")
	}
	if _, ok := deviceDisconnect.Load("d1"); ok {
		t.Error("下线后 key 应随 session 清除")
	}

	// 4. 配置 offlineTimeout：超时标记 true 但 session 保留
	dev.Config[DEVICE_TIMEOUT_KEY] = "60"
	PutSession("d1", sess, false)
	DelSessionWithTimeoutCheck("d1")
	if GetSession("d1") == nil {
		t.Fatal("超时场景 session 应保留")
	}
	if !IsDeviceDisconnect("d1") {
		t.Error("超时后应标记断开")
	}

	// 5. 超时后重新上线 → 恢复在线
	PutSession("d1", sess, false)
	if IsDeviceDisconnect("d1") {
		t.Error("重新上线后应在线")
	}

	// 6. 会话被外部（如 Redis 侧过期）清掉后兜底清理
	dev.Config[DEVICE_TIMEOUT_KEY] = "60"
	DelSessionWithTimeoutCheck("d1") // 先置 true
	PutSession("d1", sess, false)    // 再上线置 false
	defaultSessions.Delete("d1")     // 模拟外部直接删会话（绕过封装）
	if _, ok := deviceDisconnect.Load("d1"); !ok {
		t.Fatal("前置条件：key 应残留")
	}
	DelSessionWithOfflineReason("d1", "expire") // 兜底路径：Delete 返回 false 也应清 key
	if _, ok := deviceDisconnect.Load("d1"); ok {
		t.Error("兜底清理后 key 不应残留")
	}
}
