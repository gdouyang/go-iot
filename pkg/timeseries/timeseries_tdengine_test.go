package timeseries_test

import (
	"encoding/json"
	"os"
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/store"
	"go-iot/pkg/timeseries"
	"go-iot/pkg/tsl"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

// 扁平 type 字段，与 pkg/tsl 解析器一致
const text = `
{
  "events": [
    {
      "type": "object",
      "name": "fire_alarm",
      "id": "fire_alarm",
      "expands": {
        "level": "ordinary"
      },
      "properties": [
        {
          "type": "float",
          "name": "lnt",
          "id": "lnt",
          "expands": {}
        },
        {
          "type": "float",
          "name": "lat",
          "id": "lat",
          "expands": {}
        }
      ]
    }
  ],
  "properties": [
    {
      "id": "light",
      "name": "亮度",
      "type": "int",
      "unit": "",
      "expands": {
        "readOnly": "true"
      }
    },
    {
      "id": "current",
      "name": "电流",
      "type": "double",
      "scale": 2,
      "unit": "milliAmpere",
      "expands": {
        "readOnly": "true"
      }
    }
  ],
  "functions": []
}
`

// TestTdengine 基础冒烟：属性/事件/日志写读（依赖外部 TDengine）。
// 运行：TDENGINE_TEST=1 go test ./pkg/timeseries/ -run TestTdengine$ -v -count=1
// 更细用例见 timeseries_tdengine_detail_test.go / disk / schema 等。
func TestTdengine(t *testing.T) {
	logs.InitNop()
	if os.Getenv("TDENGINE_TEST") != "1" {
		t.Skip("set TDENGINE_TEST=1 to run against local TDengine (http://localhost:6041)")
	}

	ts := timeseries.TdengineTimeSeries{}
	core.RegDeviceStore(store.NewMockDeviceStore())
	product, err := core.NewProduct("td_test", map[string]string{}, core.TIME_SERISE_TDENGINE, text)
	require.NoError(t, err)
	require.NotNil(t, product)
	require.NoError(t, core.PutProduct(product))

	device := core.NewDevice("td_dev_1234", product.Id, 0)
	require.NoError(t, core.PutDevice(device))

	d := tsl.NewTslData()
	require.NoError(t, d.FromJson(text))
	product.TslData = d

	// 清理旧 super table（忽略错误）
	_ = ts.Del(product)

	require.NoError(t, ts.PublishModel(product, *d), "PublishModel should create database/stables")

	// ---- properties ----
	require.NoError(t, ts.SaveProperties(product, map[string]any{
		"deviceId": device.Id,
		"current":  10.5,
		"light":    80,
	}))
	timeseries.FlushTdengineBatch()
	{
		query := core.TimeDataSearchRequest{DeviceId: device.Id, PageNum: 1, PageSize: 10}
		res, err := ts.QueryProperty(product, query)
		require.NoError(t, err)
		raw, _ := json.Marshal(res)
		t.Logf("QueryProperty: %s", raw)
		require.GreaterOrEqual(t, res["totalCount"], int64(1), "should have property rows")
		list, ok := res["list"].([]map[string]any)
		require.True(t, ok)
		require.NotEmpty(t, list)
	}

	// ---- events ----
	require.NoError(t, ts.SaveEvents(product, "fire_alarm", map[string]any{
		"deviceId": device.Id,
		"lnt":      90.1,
		"lat":      100.2,
	}))
	timeseries.FlushTdengineBatch()
	{
		query := core.TimeDataSearchRequest{DeviceId: device.Id, PageNum: 1, PageSize: 10}
		res, err := ts.QueryEvent(product, "fire_alarm", query)
		require.NoError(t, err)
		raw, _ := json.Marshal(res)
		t.Logf("QueryEvent: %s", raw)
		require.GreaterOrEqual(t, res["totalCount"], int64(1))
	}

	// ---- logs ----
	require.NoError(t, ts.SaveLogs(product, core.LogData{
		DeviceId: device.Id,
		Content:  `{"state":"online"}`,
		Type:     "online",
	}))
	timeseries.FlushTdengineBatch()
	{
		query := core.TimeDataSearchRequest{DeviceId: device.Id, PageNum: 1, PageSize: 10}
		res, err := ts.QueryLogs(product, query)
		require.NoError(t, err)
		raw, _ := json.Marshal(res)
		t.Logf("QueryLogs: %s", raw)
		require.GreaterOrEqual(t, res["totalCount"], int64(1))
	}
}
