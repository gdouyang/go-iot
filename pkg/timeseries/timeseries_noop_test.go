package timeseries_test

import (
	"sync"
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/tsl"

	"github.com/stretchr/testify/require"
)

func TestNoopTimeSeries_RegisterAndOperations(t *testing.T) {
	ts := core.GetTimeSeries(core.TIME_SERISE_NOOP)
	require.NotNil(t, ts, "NoopTimeSeries must be registered")
	require.Equal(t, core.TIME_SERISE_NOOP, ts.Id())

	prod, err := core.NewProduct("noop-prod", map[string]string{}, core.TIME_SERISE_NOOP, `{"properties":[{"id":"temp","type":"float"}],"events":[],"functions":[]}`)
	require.NoError(t, err)

	require.NoError(t, ts.PublishModel(prod, tsl.TslData{}))
	require.NoError(t, ts.SaveProperties(prod, map[string]any{"deviceId": "d1", "temp": 25.5}))
	require.NoError(t, ts.SaveEvents(prod, "alarm", map[string]any{"deviceId": "d1"}))
	require.NoError(t, ts.SaveLogs(prod, core.LogData{DeviceId: "d1", Content: "test"}))
	require.NoError(t, ts.Del(prod))

	res, err := ts.QueryProperty(prod, core.TimeDataSearchRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, res["total"])
}

func TestNoopTimeSeries_HighConcurrencyBenchmark(t *testing.T) {
	ts := core.GetTimeSeries(core.TIME_SERISE_NOOP)
	prod, err := core.NewProduct("noop-bench", map[string]string{}, core.TIME_SERISE_NOOP, `{"properties":[],"events":[],"functions":[]}`)
	require.NoError(t, err)

	const (
		workers = 50
		ops     = 10000
	)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				_ = ts.SaveProperties(prod, map[string]any{"deviceId": "dev", "val": i})
			}
		}(w)
	}
	wg.Wait()
}
