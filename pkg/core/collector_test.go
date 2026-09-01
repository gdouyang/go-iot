package core

import (
	"testing"

	"go-iot/pkg/tsl"

	"github.com/stretchr/testify/require"
)

type stubCollectorRuntime struct {
	product string
	device  string
	cleared string
	now     string
}

func (s *stubCollectorRuntime) Reload(productId string)      { s.product = productId }
func (s *stubCollectorRuntime) ReloadDevice(deviceId string) { s.device = deviceId }
func (s *stubCollectorRuntime) Invalidate(productId string)  { s.cleared = productId }
func (s *stubCollectorRuntime) CollectNow(deviceId, groupId string) error {
	s.now = deviceId + "/" + groupId
	return nil
}

func (s *stubCollectorRuntime) Normalize(raw []byte, _ *tsl.TslData) ([]byte, error) {
	return raw, nil
}

func TestCollectorRuntimeRegistryByNetworkType(t *testing.T) {
	const net = "TEST_COLLECTOR"
	t.Cleanup(func() { RegCollectorRuntime(net, nil) })

	require.NotPanics(t, func() {
		GetCollectorRuntime("").Reload("p")
		GetCollectorRuntime("MQTT_BROKER").ReloadDevice("d")
		GetCollectorRuntime(net).Invalidate("p")
	})
	require.ErrorIs(t, GetCollectorRuntime("").CollectNow("d", ""), ErrCollectorNotActive)
	require.ErrorIs(t, GetCollectorRuntime(net).CollectNow("d", ""), ErrCollectorNotActive)

	st := &stubCollectorRuntime{}
	RegCollectorRuntime(net, st)
	GetCollectorRuntime("test_collector").Reload("p1")
	GetCollectorRuntime(net).ReloadDevice("d1")
	GetCollectorRuntime(net).Invalidate("p2")
	require.NoError(t, GetCollectorRuntime(net).CollectNow("d2", "g1"))
	require.Equal(t, "p1", st.product)
	require.Equal(t, "d1", st.device)
	require.Equal(t, "p2", st.cleared)
	require.Equal(t, "d2/g1", st.now)

	other := &stubCollectorRuntime{}
	RegCollectorRuntime("OTHER_COLLECTOR", other)
	t.Cleanup(func() { RegCollectorRuntime("OTHER_COLLECTOR", nil) })
	GetCollectorRuntime("OTHER_COLLECTOR").Reload("px")
	require.Equal(t, "px", other.product)
	require.Equal(t, "p1", st.product)

	RegCollectorRuntime(net, nil)
	require.ErrorIs(t, GetCollectorRuntime(net).CollectNow("d", ""), ErrCollectorNotActive)
}

func TestNoopCollectorNormalize(t *testing.T) {
	_, err := GetCollectorRuntime("MQTT_BROKER").Normalize(nil, nil)
	require.NoError(t, err)
	_, err = GetCollectorRuntime("").Normalize([]byte(`{}`), nil)
	require.NoError(t, err)
	_, err = GetCollectorRuntime("MQTT_BROKER").Normalize([]byte(`{"enabled":true}`), nil)
	require.ErrorIs(t, err, ErrCollectorNotSupported)
}

func TestLoadCollectorJSONLoader(t *testing.T) {
	t.Cleanup(func() { SetCollectorJSONLoader(nil) })
	raw, err := LoadCollectorJSON("p1")
	require.NoError(t, err)
	require.Equal(t, "", raw)

	SetCollectorJSONLoader(func(productId string) (string, error) {
		return "cfg-" + productId, nil
	})
	raw, err = LoadCollectorJSON("p1")
	require.NoError(t, err)
	require.Equal(t, "cfg-p1", raw)
	raw, err = LoadCollectorJSON("")
	require.NoError(t, err)
	require.Equal(t, "", raw)
}
