package timeseries_test

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/store"
	"go-iot/pkg/timeseries"
	"go-iot/pkg/tsl"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

// 更丰富的 TSL：多类型 + object + 事件 + 日志
const tdDetailTSL = `
{
  "events": [
    {
      "type": "object",
      "name": "fire_alarm",
      "id": "fire_alarm",
      "properties": [
        {"type": "float", "name": "lnt", "id": "lnt"},
        {"type": "float", "name": "lat", "id": "lat"}
      ]
    },
    {
      "type": "string",
      "name": "simple",
      "id": "simple_evt"
    }
  ],
  "properties": [
    {"id": "light", "name": "亮度", "type": "int"},
    {"id": "current", "name": "电流", "type": "double"},
    {"id": "flag", "name": "开关", "type": "bool"},
    {"id": "mode", "name": "模式", "type": "string"},
    {"id": "cnt", "name": "计数", "type": "long"},
    {
      "id": "obj",
      "name": "对象",
      "type": "object",
      "properties": [
        {"id": "name", "name": "名称", "type": "string"},
        {"id": "v", "name": "值", "type": "int"}
      ]
    }
  ],
  "functions": []
}
`

func requireTdengine(t *testing.T) {
	t.Helper()
	logs.InitNop()
	if os.Getenv("TDENGINE_TEST") != "1" {
		t.Skip("set TDENGINE_TEST=1 to run against local TDengine (http://localhost:6041)")
	}
}

func setupTdProduct(t *testing.T, productID string) (*timeseries.TdengineTimeSeries, *core.Product) {
	t.Helper()
	requireTdengine(t)
	core.RegDeviceStore(store.NewMockDeviceStore())

	ts := &timeseries.TdengineTimeSeries{}
	product, err := core.NewProduct(productID, map[string]string{}, core.TIME_SERISE_TDENGINE, tdDetailTSL)
	require.NoError(t, err)
	require.NotNil(t, product)
	require.NoError(t, core.PutProduct(product))

	d := tsl.NewTslData()
	require.NoError(t, d.FromJson(tdDetailTSL))
	product.TslData = d

	_ = ts.Del(product)
	require.NoError(t, ts.PublishModel(product, *d))
	return ts, product
}

func flushTd(t *testing.T) {
	t.Helper()
	timeseries.FlushTdengineBatch()
	// 给 REST 一点时间完成
	time.Sleep(50 * time.Millisecond)
}

func asInt64(t *testing.T, v any) int64 {
	t.Helper()
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		t.Fatalf("unexpected numeric type %T", v)
		return 0
	}
}

func TestTdengineConnectivity(t *testing.T) {
	requireTdengine(t)
	ts := timeseries.TdengineTimeSeries{}
	// 通过 PublishModel 建库验证连通
	p, err := core.NewProduct("td_conn", map[string]string{}, core.TIME_SERISE_TDENGINE, `{"properties":[{"id":"x","name":"x","type":"int"}],"events":[],"functions":[]}`)
	require.NoError(t, err)
	d := tsl.NewTslData()
	require.NoError(t, d.FromJson(`{"properties":[{"id":"x","name":"x","type":"int"}],"events":[],"functions":[]}`))
	p.TslData = d
	require.NoError(t, ts.PublishModel(p, *d))
}

func TestTdengineSaveValidation(t *testing.T) {
	ts, product := setupTdProduct(t, "td_valid")

	t.Run("missing deviceId", func(t *testing.T) {
		err := ts.SaveProperties(product, map[string]any{"light": 1})
		require.Error(t, err)
		require.Contains(t, err.Error(), "deviceId")
	})
	t.Run("empty property payload", func(t *testing.T) {
		err := ts.SaveProperties(product, map[string]any{"deviceId": "d1", "unknown": 1})
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty")
	})
	t.Run("query without deviceId", func(t *testing.T) {
		_, err := ts.QueryProperty(product, core.TimeDataSearchRequest{})
		require.Error(t, err)
	})
	t.Run("unknown event", func(t *testing.T) {
		err := ts.SaveEvents(product, "no_such", map[string]any{"deviceId": "d1", "x": 1})
		require.Error(t, err)
	})
	t.Run("logs missing deviceId", func(t *testing.T) {
		err := ts.SaveLogs(product, core.LogData{Content: "x", Type: "t"})
		require.Error(t, err)
	})
}

func TestTdengineMultiTypeAndObject(t *testing.T) {
	ts, product := setupTdProduct(t, "td_types")
	devID := "td_types_dev1"
	require.NoError(t, core.PutDevice(core.NewDevice(devID, product.GetId(), 0)))

	// 标量类型（object 嵌套 map 当前不支持直接写入 obj_ 列，需扁平 key 且在 PropertiesMap 中）
	require.NoError(t, ts.SaveProperties(product, map[string]any{
		"deviceId": devID,
		"light":    42,
		"current":  3.14,
		"flag":     true,
		"mode":     "auto",
		"cnt":      int64(1000001),
	}))
	flushTd(t)

	res, err := ts.QueryProperty(product, core.TimeDataSearchRequest{DeviceId: devID, PageNum: 1, PageSize: 20})
	require.NoError(t, err)
	require.GreaterOrEqual(t, asInt64(t, res["totalCount"]), int64(1))
	list := res["list"].([]map[string]any)
	require.NotEmpty(t, list)
	row := list[0]
	require.EqualValues(t, 42, row["light"])
	require.InDelta(t, 3.14, toFloat(t, row["current"]), 0.001)
	require.Equal(t, true, row["flag"])
	require.Equal(t, "auto", row["mode"])
	t.Logf("rows=%d first=%v", len(list), row)
}

func toFloat(t *testing.T, v any) float64 {
	t.Helper()
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		t.Fatalf("not float: %T", v)
		return 0
	}
}

func TestTdenginePagination(t *testing.T) {
	ts, product := setupTdProduct(t, "td_page")
	devID := "td_page_dev"
	require.NoError(t, core.PutDevice(core.NewDevice(devID, product.GetId(), 0)))

	const n = 25
	for i := 0; i < n; i++ {
		require.NoError(t, ts.SaveProperties(product, map[string]any{
			"deviceId": devID,
			"light":    i,
			"current":  float64(i) + 0.1,
		}))
	}
	flushTd(t)

	page1, err := ts.QueryProperty(product, core.TimeDataSearchRequest{DeviceId: devID, PageNum: 1, PageSize: 10})
	require.NoError(t, err)
	total := asInt64(t, page1["totalCount"])
	require.EqualValues(t, n, total, "timestamp collisions reduced count")
	require.Equal(t, 10, page1["pageSize"])
	require.Equal(t, 1, page1["pageNum"])
	totalPage := int((total + 9) / 10)
	require.Equal(t, totalPage, page1["totalPage"])
	require.Equal(t, true, page1["firstPage"])
	require.Equal(t, totalPage == 1, page1["lastPage"])
	require.Len(t, page1["list"].([]map[string]any), 10)

	lastPage, err := ts.QueryProperty(product, core.TimeDataSearchRequest{DeviceId: devID, PageNum: totalPage, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, true, lastPage["lastPage"])
	remain := int(total) - (totalPage-1)*10
	require.Len(t, lastPage["list"].([]map[string]any), remain)
}

func TestTdengineTimeRangeFilter(t *testing.T) {
	ts, product := setupTdProduct(t, "td_range")
	devID := "td_range_dev"
	require.NoError(t, core.PutDevice(core.NewDevice(devID, product.GetId(), 0)))

	// 写入若干点
	for i := 0; i < 5; i++ {
		require.NoError(t, ts.SaveProperties(product, map[string]any{
			"deviceId": devID,
			"light":    100 + i,
		}))
	}
	flushTd(t)

	// 全量
	all, err := ts.QueryProperty(product, core.TimeDataSearchRequest{DeviceId: devID, PageSize: 100})
	require.NoError(t, err)
	total := asInt64(t, all["totalCount"])
	require.GreaterOrEqual(t, total, int64(5))

	// GTE 用较早时间，应能查出
	past := time.Now().Add(-1 * time.Hour).Format("2006-01-02 15:04:05.000")
	res, err := ts.QueryProperty(product, core.TimeDataSearchRequest{
		DeviceId: devID,
		PageSize: 100,
		Condition: []core.SearchTerm{
			{Key: "createTime", Value: past, Oper: core.GTE},
		},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, asInt64(t, res["totalCount"]), int64(5))

	// 未来时间 LT 边界：应很少或为 0 取决于数据；GT 未来应 0
	future := time.Now().Add(1 * time.Hour).Format("2006-01-02 15:04:05.000")
	res2, err := ts.QueryProperty(product, core.TimeDataSearchRequest{
		DeviceId: devID,
		PageSize: 100,
		Condition: []core.SearchTerm{
			{Key: "createTime", Value: future, Oper: core.GT},
		},
	})
	require.NoError(t, err)
	require.EqualValues(t, 0, asInt64(t, res2["totalCount"]))

	// BTW 覆盖最近一小时
	res3, err := ts.QueryProperty(product, core.TimeDataSearchRequest{
		DeviceId: devID,
		PageSize: 100,
		Condition: []core.SearchTerm{
			{Key: "createTime", Value: []any{past, future}, Oper: core.BTW},
		},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, asInt64(t, res3["totalCount"]), int64(5))
}

func TestTdengineEventsAndLogsDetailed(t *testing.T) {
	ts, product := setupTdProduct(t, "td_evtlog")
	dFire, dSimple, dLog := "td_evtlog_fire", "td_evtlog_simple", "td_evtlog_log"
	require.NoError(t, core.PutDevice(core.NewDevice(dFire, product.GetId(), 0)))
	require.NoError(t, core.PutDevice(core.NewDevice(dSimple, product.GetId(), 0)))
	require.NoError(t, core.PutDevice(core.NewDevice(dLog, product.GetId(), 0)))

	require.NoError(t, ts.SaveEvents(product, "fire_alarm", map[string]any{
		"deviceId": dFire, "lnt": 12.3, "lat": 45.6,
	}))
	// 同一设备写不同 event 子表：insertFragment 已按 stable 区分子表名
	require.NoError(t, ts.SaveEvents(product, "simple_evt", map[string]any{
		"deviceId": dFire, "simple_evt": "hello",
	}))
	require.NoError(t, ts.SaveEvents(product, "simple_evt", map[string]any{
		"deviceId": dSimple, "simple_evt": "world",
	}))
	require.NoError(t, ts.SaveLogs(product, core.LogData{
		DeviceId: dLog, Type: "online", Content: `{"ok":true}`,
	}))
	require.NoError(t, ts.SaveLogs(product, core.LogData{
		DeviceId: dLog, Type: "offline", Content: `{"ok":false}`,
	}))
	flushTd(t)

	ev, err := ts.QueryEvent(product, "fire_alarm", core.TimeDataSearchRequest{DeviceId: dFire, PageSize: 10})
	require.NoError(t, err)
	require.GreaterOrEqual(t, asInt64(t, ev["totalCount"]), int64(1))

	ev2, err := ts.QueryEvent(product, "simple_evt", core.TimeDataSearchRequest{DeviceId: dFire, PageSize: 10})
	require.NoError(t, err)
	require.GreaterOrEqual(t, asInt64(t, ev2["totalCount"]), int64(1))

	logsRes, err := ts.QueryLogs(product, core.TimeDataSearchRequest{
		DeviceId: dLog,
		PageSize: 10,
		Condition: []core.SearchTerm{
			{Key: "type", Value: "online", Oper: core.EQ},
		},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, asInt64(t, logsRes["totalCount"]), int64(1))
}

func TestTdengineMultiDeviceIsolation(t *testing.T) {
	ts, product := setupTdProduct(t, "td_iso")
	d1, d2 := "td_iso_a", "td_iso_b"
	require.NoError(t, core.PutDevice(core.NewDevice(d1, product.GetId(), 0)))
	require.NoError(t, core.PutDevice(core.NewDevice(d2, product.GetId(), 0)))

	require.NoError(t, ts.SaveProperties(product, map[string]any{"deviceId": d1, "light": 1}))
	require.NoError(t, ts.SaveProperties(product, map[string]any{"deviceId": d2, "light": 2}))
	require.NoError(t, ts.SaveProperties(product, map[string]any{"deviceId": d1, "light": 3}))
	flushTd(t)

	r1, err := ts.QueryProperty(product, core.TimeDataSearchRequest{DeviceId: d1, PageSize: 100})
	require.NoError(t, err)
	require.EqualValues(t, 2, asInt64(t, r1["totalCount"]))

	r2, err := ts.QueryProperty(product, core.TimeDataSearchRequest{DeviceId: d2, PageSize: 100})
	require.NoError(t, err)
	require.EqualValues(t, 1, asInt64(t, r2["totalCount"]))
}

func TestTdengineRepublishIdempotent(t *testing.T) {
	ts, product := setupTdProduct(t, "td_repub")
	// 重复 PublishModel 应 IF NOT EXISTS 成功
	require.NoError(t, ts.PublishModel(product, *product.GetTsl()))
	require.NoError(t, ts.PublishModel(product, *product.GetTsl()))
}

// ---------- 压力测试 ----------
// TDENGINE_STRESS=1 开启
// 可选：TDENGINE_STRESS_N（默认 2000）、TDENGINE_STRESS_CONCURRENCY（默认 20）

func stressParams(t *testing.T) (n, concurrency int) {
	t.Helper()
	if os.Getenv("TDENGINE_STRESS") != "1" {
		t.Skip("set TDENGINE_STRESS=1 (and TDENGINE_TEST=1) for stress tests")
	}
	requireTdengine(t)
	n = 2000
	concurrency = 20
	if v := os.Getenv("TDENGINE_STRESS_N"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			n = x
		}
	}
	if v := os.Getenv("TDENGINE_STRESS_CONCURRENCY"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			concurrency = x
		}
	}
	return n, concurrency
}

func TestTdengineStressSequentialWrite(t *testing.T) {
	n, _ := stressParams(t)
	ts, product := setupTdProduct(t, "td_stress_seq")
	devID := "td_stress_seq_dev"
	require.NoError(t, core.PutDevice(core.NewDevice(devID, product.GetId(), 0)))

	start := time.Now()
	var fail int64
	for i := 0; i < n; i++ {
		err := ts.SaveProperties(product, map[string]any{
			"deviceId": devID,
			"light":    i % 1000,
			"current":  float64(i) * 0.01,
		})
		if err != nil {
			atomic.AddInt64(&fail, 1)
		}
	}
	flushTd(t)
	elapsed := time.Since(start)
	qps := float64(n) / elapsed.Seconds()

	res, err := ts.QueryProperty(product, core.TimeDataSearchRequest{DeviceId: devID, PageSize: 1})
	require.NoError(t, err)
	got := asInt64(t, res["totalCount"])

	t.Logf("sequential write(batch): n=%d fail=%d elapsed=%s qps=%.1f stored=%d", n, fail, elapsed, qps, got)
	require.EqualValues(t, 0, fail)
	require.GreaterOrEqual(t, got, int64(float64(n)*0.95), "stored rows too low")
}

func TestTdengineStressConcurrentMultiDevice(t *testing.T) {
	n, concurrency := stressParams(t)
	ts, product := setupTdProduct(t, "td_stress_cc")

	// 每协程独立 device，避免同表同 ts 冲突
	type job struct{ i int }
	jobs := make(chan job, n)
	for i := 0; i < n; i++ {
		jobs <- job{i: i}
	}
	close(jobs)

	var fail, okCnt int64
	start := time.Now()
	var wg sync.WaitGroup
	for c := 0; c < concurrency; c++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := range jobs {
				devID := fmt.Sprintf("tdcc_w%d_d%d", worker, j.i%200) // 200 设备轮转
				// 确保设备存在（重复 Put 可接受）
				_ = core.PutDevice(core.NewDevice(devID, product.GetId(), 0))
				err := ts.SaveProperties(product, map[string]any{
					"deviceId": devID,
					"light":    j.i,
					"current":  float64(j.i) * 0.1,
					"mode":     "stress",
				})
				if err != nil {
					atomic.AddInt64(&fail, 1)
				} else {
					atomic.AddInt64(&okCnt, 1)
				}
			}
		}(c)
	}
	wg.Wait()
	flushTd(t)
	elapsed := time.Since(start)
	qps := float64(n) / elapsed.Seconds()

	// 抽样查询一个设备
	sample := "tdcc_w0_d0"
	res, err := ts.QueryProperty(product, core.TimeDataSearchRequest{DeviceId: sample, PageSize: 1})
	require.NoError(t, err)

	t.Logf("concurrent multi-device(batch): n=%d concurrency=%d ok=%d fail=%d elapsed=%s qps=%.1f sampleTotal=%v",
		n, concurrency, okCnt, fail, elapsed, qps, res["totalCount"])
	require.EqualValues(t, 0, fail)
	require.EqualValues(t, n, okCnt)
}

func TestTdengineStressConcurrentSameDevice(t *testing.T) {
	n, concurrency := stressParams(t)
	// 同设备并发写：REST 串行化后时间戳可能撞车，重点观察错误率与最终条数
	if n > 1000 {
		n = 1000 // 同设备压测默认收紧，避免过久
	}
	ts, product := setupTdProduct(t, "td_stress_same")
	devID := "td_stress_same_dev"
	require.NoError(t, core.PutDevice(core.NewDevice(devID, product.GetId(), 0)))

	var fail, okCnt int64
	start := time.Now()
	var wg sync.WaitGroup
	per := n / concurrency
	if per < 1 {
		per = 1
	}
	for c := 0; c < concurrency; c++ {
		wg.Add(1)
		go func(worker, count int) {
			defer wg.Done()
			for i := 0; i < count; i++ {
				err := ts.SaveProperties(product, map[string]any{
					"deviceId": devID,
					"light":    worker*10000 + i,
					"current":  float64(i),
				})
				if err != nil {
					atomic.AddInt64(&fail, 1)
				} else {
					atomic.AddInt64(&okCnt, 1)
				}
			}
		}(c, per)
	}
	wg.Wait()
	flushTd(t)
	elapsed := time.Since(start)
	totalAttempt := int64(per * concurrency)

	res, err := ts.QueryProperty(product, core.TimeDataSearchRequest{DeviceId: devID, PageSize: 1})
	require.NoError(t, err)
	stored := asInt64(t, res["totalCount"])
	qps := float64(totalAttempt) / elapsed.Seconds()

	t.Logf("concurrent same-device(batch): attempt=%d ok=%d fail=%d stored=%d elapsed=%s qps=%.1f",
		totalAttempt, okCnt, fail, stored, elapsed, qps)
	require.Greater(t, stored, int64(0))
	// 单调 createTime 后同设备并发应接近全量
	require.GreaterOrEqual(t, stored, int64(float64(totalAttempt)*0.9))
}

func TestTdengineStressQueryUnderLoad(t *testing.T) {
	n, concurrency := stressParams(t)
	if n > 1500 {
		n = 1500
	}
	ts, product := setupTdProduct(t, "td_stress_q")
	devID := "td_stress_q_dev"
	require.NoError(t, core.PutDevice(core.NewDevice(devID, product.GetId(), 0)))

	// 预热写入
	for i := 0; i < n; i++ {
		require.NoError(t, ts.SaveProperties(product, map[string]any{
			"deviceId": devID,
			"light":    i,
		}))
	}
	flushTd(t)

	var qFail, qOk int64
	start := time.Now()
	var wg sync.WaitGroup
	rounds := concurrency * 20
	jobs := make(chan int, rounds)
	for i := 0; i < rounds; i++ {
		jobs <- i
	}
	close(jobs)
	for c := 0; c < concurrency; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				_, err := ts.QueryProperty(product, core.TimeDataSearchRequest{
					DeviceId: devID,
					PageNum:  1,
					PageSize: 20,
				})
				if err != nil {
					atomic.AddInt64(&qFail, 1)
				} else {
					atomic.AddInt64(&qOk, 1)
				}
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	qps := float64(rounds) / elapsed.Seconds()
	t.Logf("query under load: rounds=%d ok=%d fail=%d elapsed=%s qps=%.1f", rounds, qOk, qFail, elapsed, qps)
	require.EqualValues(t, 0, qFail)
	require.EqualValues(t, rounds, qOk)
}
