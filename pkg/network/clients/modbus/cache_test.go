package modbus

import (
	"testing"

	"go-iot/pkg/core"

	"github.com/stretchr/testify/require"
)

func TestCollectorCacheRoundTrip(t *testing.T) {
	t.Cleanup(func() {
		core.SetCollectorJSONLoader(nil)
		collectorCache.Delete("p1")
	})
	core.SetCollectorJSONLoader(func(productId string) (string, error) {
		return `{"enabled":true,"groups":[{"id":"g1","intervalMs":1000}],"points":[{"id":"p","propertyId":"Temperature","groupId":"g1"}]}`, nil
	})
	cfg := GetCollector("p1")
	require.True(t, cfg.Enabled)
	require.Len(t, cfg.Groups, 1)

	PutCollector("p1", &CollectorConfig{Enabled: false})
	got := GetCollector("p1")
	require.False(t, got.Enabled)

	DeleteCollector("p1")
	got = GetCollector("p1")
	require.True(t, got.Enabled)
}

func TestCollectorRuntimeCollectNowNoSession(t *testing.T) {
	err := modbusCollectorRuntime{}.CollectNow("no-such-device", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "设备未连接")
}
