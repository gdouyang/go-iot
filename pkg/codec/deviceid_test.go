package codec

import (
	"testing"

	"go-iot/pkg/core"

	"github.com/stretchr/testify/require"
)

func TestExtractDeviceIdFromContext(t *testing.T) {
	require.Equal(t, "", extractDeviceId(nil))
	require.Equal(t, "d1", extractDeviceId(&core.BaseContext{DeviceId: "d1"}))
	require.Equal(t, "d2", extractDeviceId(core.BaseContext{DeviceId: " d2 "}))
	require.Equal(t, "d3", extractDeviceId(core.FuncInvokeContext{
		BaseContext: core.BaseContext{DeviceId: "d3"},
	}))
}

type stubSession struct {
	id string
}

func (s *stubSession) Disconnect() error          { return nil }
func (s *stubSession) GetDeviceId() string        { return s.id }
func (s *stubSession) SetDeviceId(id string)      { s.id = id }
func (s *stubSession) Close() error               { return nil }
func (s *stubSession) GetConInfo() map[string]any { return nil }

func TestExtractDeviceIdFromSession(t *testing.T) {
	ctx := &core.BaseContext{
		DeviceId: "",
		Session:  &stubSession{id: "sess-1"},
	}
	require.Equal(t, "sess-1", extractDeviceId(ctx))
}
