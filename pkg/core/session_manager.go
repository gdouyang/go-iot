package core

import "sync"

// SessionManager 设备连接会话表。接口化便于 App 注入与单测替换。
// 领域副作用（上线事件、离线命令）仍由 PutSession / DelSession* 包级函数处理。
type SessionManager interface {
	Get(deviceId string) Session
	Put(deviceId string, session Session)
	// Delete 删除并返回此前是否存在。
	Delete(deviceId string) bool
}

// MemorySessionManager 进程内 sync.Map 实现。
type MemorySessionManager struct {
	m sync.Map
}

func NewMemorySessionManager() *MemorySessionManager {
	return &MemorySessionManager{}
}

func (s *MemorySessionManager) Get(deviceId string) Session {
	if val, ok := s.m.Load(deviceId); ok {
		return val.(Session)
	}
	return nil
}

func (s *MemorySessionManager) Put(deviceId string, session Session) {
	s.m.Store(deviceId, session)
}

func (s *MemorySessionManager) Delete(deviceId string) bool {
	_, ok := s.m.LoadAndDelete(deviceId)
	return ok
}

// package default — 过渡期兼容；App.New 会 RegSessionManager。
var defaultSessions SessionManager = NewMemorySessionManager()

// RegSessionManager 注册会话管理器（可替换实现）。
func RegSessionManager(m SessionManager) {
	if m == nil {
		return
	}
	defaultSessions = m
}

// GetSessionManager 返回当前会话管理器。
func GetSessionManager() SessionManager {
	return defaultSessions
}
