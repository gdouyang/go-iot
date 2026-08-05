package core_test

import (
	"testing"

	"go-iot/pkg/core"

	"github.com/stretchr/testify/require"
)

type stubSession struct {
	id      string
	conInfo map[string]any
}

func (s *stubSession) Disconnect() error          { return nil }
func (s *stubSession) Close() error               { return nil }
func (s *stubSession) GetDeviceId() string        { return s.id }
func (s *stubSession) GetConInfo() map[string]any { return s.conInfo }
func (s *stubSession) SetDeviceId(id string)      { s.id = id }

func TestMemorySessionManager_GetPutDelete(t *testing.T) {
	m := core.NewMemorySessionManager()
	require.Nil(t, m.Get("d1"))

	s := &stubSession{id: "d1", conInfo: map[string]any{}}
	m.Put("d1", s)
	require.Equal(t, s, m.Get("d1"))

	require.True(t, m.Delete("d1"))
	require.Nil(t, m.Get("d1"))
	require.False(t, m.Delete("d1"))
}

func TestRegSessionManager_ReplacesPackageDefault(t *testing.T) {
	prev := core.GetSessionManager()
	t.Cleanup(func() { core.RegSessionManager(prev) })

	m := core.NewMemorySessionManager()
	core.RegSessionManager(m)
	require.Equal(t, m, core.GetSessionManager())

	s := &stubSession{id: "x", conInfo: map[string]any{}}
	m.Put("x", s)
	require.Equal(t, s, core.GetSession("x"))
}

func TestMemoryOfflineCommandQueue_EnqueueAndTakeAll(t *testing.T) {
	q := core.NewMemoryOfflineCommandQueue()
	msg := core.FuncInvoke{DeviceId: "dev-1", FunctionId: "f1"}
	err := q.Enqueue(msg)
	require.NotNil(t, err)
	require.Equal(t, 200, err.Code)

	// OTA rejected
	ota := core.FuncInvoke{DeviceId: "dev-1", FunctionId: core.OTA_UPDATE}
	require.Equal(t, 400, q.Enqueue(ota).Code)

	// fill to max
	for i := 0; i < 19; i++ {
		require.Equal(t, 200, q.Enqueue(core.FuncInvoke{DeviceId: "dev-1", FunctionId: "f"}).Code)
	}
	require.Equal(t, 400, q.Enqueue(core.FuncInvoke{DeviceId: "dev-1", FunctionId: "overflow"}).Code)

	all := q.TakeAll("dev-1")
	require.Len(t, all, 20)
	require.Empty(t, q.TakeAll("dev-1"))
}
