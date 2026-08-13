package timeseries_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

// boardcollector 物模型字段（对齐 C:\Users\oyang\Downloads\boardcollector.json）
// latitude,longitude double; velocity,Pitch,Yaw,Roll,height float; millisecond long
//
// ES 对照索引 goiot-properties-boardcollector-202608: docs≈2114494, store≈220.7mb
//
// 运行：
//
//	TDENGINE_TEST=1 TDENGINE_DISK=1 go test ./pkg/timeseries/ -run TestTdengineDiskBoardcollector -v -count=1 -timeout 1h
//
// 可选：TDENGINE_DISK_N（默认 2114494）、TDENGINE_DISK_BATCH（默认 2000）、TDENGINE_DOCKER
func TestTdengineDiskBoardcollector(t *testing.T) {
	logs.InitNop()
	if os.Getenv("TDENGINE_TEST") != "1" || os.Getenv("TDENGINE_DISK") != "1" {
		t.Skip("set TDENGINE_TEST=1 TDENGINE_DISK=1 to run boardcollector disk benchmark")
	}

	// 与 ES 月索引 docs.count 对齐
	n := 2_114_494
	batchSize := 2000
	if v := envInt("TDENGINE_DISK_N"); v > 0 {
		n = v
	}
	if v := envInt("TDENGINE_DISK_BATCH"); v > 0 {
		batchSize = v
	}

	deviceID := "boardcollector_dev1"
	stable := "goiot.properties_boardcollector"
	child := "goiot.properties_boardcollector_dev1"

	t.Log("clearing database goiot ...")
	mustTaosOK(t, "DROP DATABASE IF EXISTS goiot")
	mustTaosOK(t, "CREATE DATABASE IF NOT EXISTS goiot PRECISION 'ms' KEEP 3650d COMP 2")

	// 列名规则与 TdengineTimeSeries.columnNameRewrite 一致：snake + _
	createSQL := `CREATE STABLE IF NOT EXISTS ` + stable + ` (
  create_time_ TIMESTAMP,
  latitude_ DOUBLE,
  longitude_ DOUBLE,
  velocity_ FLOAT,
  pitch_ FLOAT,
  yaw_ FLOAT,
  roll_ FLOAT,
  height_ FLOAT,
  millisecond_ BIGINT
) TAGS (device_id_ NCHAR(64));`
	mustTaosOK(t, createSQL)

	beforeLib, err := dockerDu("/var/lib/taos")
	require.NoError(t, err)
	beforeLog, _ := dockerDu("/var/log/taos")
	t.Logf("BEFORE /var/lib/taos=%s /var/log/taos=%s", beforeLib, beforeLog)
	if u, e := queryDiskUsage("goiot"); e == nil {
		t.Logf("BEFORE ins_disk_usage: %s", compactJSON(u))
	}

	// 单设备、时间严格递增（每行 +1ms），顺序批量写入
	// 起点对齐 ES 月索引所在月
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
	t.Logf("schema=boardcollector(8 props) device=%s n=%d batch=%d base=%s (+1ms/row)",
		deviceID, n, batchSize, base.Format(time.RFC3339))

	var failBatches, okRows, okBatches int
	start := time.Now()
	for off := 0; off < n; off += batchSize {
		c := batchSize
		if off+c > n {
			c = n - off
		}
		sql := buildBoardcollectorBatch(stable, child, deviceID, off, c, base)
		if err := taosExecOK(sql); err != nil {
			failBatches++
			t.Logf("batch fail at off=%d: %v", off, err)
			continue
		}
		okBatches++
		okRows += c
	}
	writeElapsed := time.Since(start)

	_, _ = taosSQL("FLUSH DATABASE goiot")
	time.Sleep(3 * time.Second)

	afterLib, err := dockerDu("/var/lib/taos")
	require.NoError(t, err)
	afterLog, _ := dockerDu("/var/log/taos")
	totalRows, err := taosCount("select count(*) from " + stable)
	require.NoError(t, err)

	beforeB := parseDuBytes(beforeLib)
	afterB := parseDuBytes(afterLib)
	delta := afterB - beforeB
	qps := float64(okRows) / writeElapsed.Seconds()
	var bpr float64
	if totalRows > 0 && delta > 0 {
		bpr = float64(delta) / float64(totalRows)
	}

	// ES 对照
	const esDocs = 2_114_494
	esMB := 220.7
	esBytes := int64(esMB * 1024 * 1024) // 220.7mb
	esBPR := float64(esBytes) / float64(esDocs)

	t.Logf("========== boardcollector 磁盘对比压测 ==========")
	t.Logf("TSL fields(8): latitude,longitude(double) velocity,Pitch,Yaw,Roll,height(float) millisecond(long)")
	t.Logf("mode: single-device time-monotonic(+1ms) batch INSERT")
	t.Logf("target_n=%d (ES docs=%d) batch=%d", n, esDocs, batchSize)
	t.Logf("write: okRows=%d okBatches=%d failBatches=%d elapsed=%s qps=%.1f",
		okRows, okBatches, failBatches, writeElapsed, qps)
	t.Logf("count(*) on stable: %d", totalRows)
	t.Logf("TDengine /var/lib/taos: %s -> %s (delta≈%s / %d bytes)", beforeLib, afterLib, humanBytes(delta), delta)
	t.Logf("TDengine /var/log/taos: %s -> %s", beforeLog, afterLog)
	if u, e := queryDiskUsage("goiot"); e == nil {
		t.Logf("AFTER ins_disk_usage: %s", compactJSON(u))
	}
	if bpr > 0 {
		t.Logf("TDengine approx disk/row: %.1f bytes", bpr)
	}
	t.Logf("ES goiot-properties-boardcollector-202608: docs=%d store=220.7mb ≈ %.1f bytes/doc", esDocs, esBPR)
	if bpr > 0 {
		t.Logf("ratio TD/ES bytes-per-row: %.2fx", bpr/esBPR)
		// 同量级 211 万条 ES 体积 vs TD 实测
		t.Logf("same count: ES≈220.7 MiB, TD≈%s", humanBytes(delta))
	}
	t.Logf("=================================================")

	require.Equal(t, 0, failBatches)
	require.EqualValues(t, n, totalRows)
}

// TestTdengineDiskTempOnly10M 单列 temperature(float)，单设备时间递增，1000 万条磁盘占用。
//
//	TDENGINE_TEST=1 TDENGINE_DISK=1 go test ./pkg/timeseries/ -run TestTdengineDiskTempOnly10M -v -count=1 -timeout 1h
//
// 可选：TDENGINE_DISK_N=10000000 TDENGINE_DISK_BATCH=5000
func TestTdengineDiskTempOnly10M(t *testing.T) {
	logs.InitNop()
	if os.Getenv("TDENGINE_TEST") != "1" || os.Getenv("TDENGINE_DISK") != "1" {
		t.Skip("set TDENGINE_TEST=1 TDENGINE_DISK=1 to run temp-only 10M disk benchmark")
	}

	n := 10_000_000
	batchSize := 5000
	if v := envInt("TDENGINE_DISK_N"); v > 0 {
		n = v
	}
	if v := envInt("TDENGINE_DISK_BATCH"); v > 0 {
		batchSize = v
	}

	deviceID := "temp_only_dev1"
	stable := "goiot.properties_temp_only"
	child := "goiot.properties_temp_only_dev1"

	t.Log("clearing database goiot ...")
	mustTaosOK(t, "DROP DATABASE IF EXISTS goiot")
	mustTaosOK(t, "CREATE DATABASE IF NOT EXISTS goiot PRECISION 'ms' KEEP 3650d COMP 2")

	// 仅 temperature float + createTime + tag deviceId
	createSQL := `CREATE STABLE IF NOT EXISTS ` + stable + ` (
  create_time_ TIMESTAMP,
  temperature_ FLOAT
) TAGS (device_id_ NCHAR(64));`
	mustTaosOK(t, createSQL)

	beforeLib, err := dockerDu("/var/lib/taos")
	require.NoError(t, err)
	beforeLog, _ := dockerDu("/var/log/taos")
	t.Logf("BEFORE /var/lib/taos=%s /var/log/taos=%s", beforeLib, beforeLog)

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t.Logf("schema=temperature(float only) device=%s n=%d batch=%d +1ms/row", deviceID, n, batchSize)

	var failBatches, okRows, okBatches int
	start := time.Now()
	for off := 0; off < n; off += batchSize {
		c := batchSize
		if off+c > n {
			c = n - off
		}
		sql := buildTempOnlyBatch(stable, child, deviceID, off, c, base)
		if err := taosExecOK(sql); err != nil {
			failBatches++
			t.Logf("batch fail at off=%d: %v", off, err)
			continue
		}
		okBatches++
		okRows += c
	}
	writeElapsed := time.Since(start)

	_, _ = taosSQL("FLUSH DATABASE goiot")
	time.Sleep(3 * time.Second)

	afterLib, err := dockerDu("/var/lib/taos")
	require.NoError(t, err)
	afterLog, _ := dockerDu("/var/log/taos")
	totalRows, err := taosCount("select count(*) from " + stable)
	require.NoError(t, err)

	beforeB := parseDuBytes(beforeLib)
	afterB := parseDuBytes(afterLib)
	delta := afterB - beforeB
	qps := float64(okRows) / writeElapsed.Seconds()
	var bpr float64
	if totalRows > 0 && delta > 0 {
		bpr = float64(delta) / float64(totalRows)
	}

	t.Logf("========== 单列 temperature 1000万 磁盘压测 ==========")
	t.Logf("schema: create_time_ + temperature_(FLOAT) + TAG device_id_")
	t.Logf("mode: single-device time-monotonic(+1ms) batch INSERT COMP=2")
	t.Logf("target_n=%d batch=%d", n, batchSize)
	t.Logf("write: okRows=%d okBatches=%d failBatches=%d elapsed=%s qps=%.1f",
		okRows, okBatches, failBatches, writeElapsed, qps)
	t.Logf("count(*) on stable: %d", totalRows)
	t.Logf("/var/lib/taos: %s -> %s (delta≈%s / %d bytes)", beforeLib, afterLib, humanBytes(delta), delta)
	t.Logf("/var/log/taos: %s -> %s", beforeLog, afterLog)
	if u, e := queryDiskUsage("goiot"); e == nil {
		t.Logf("ins_disk_usage: %s", compactJSON(u))
	}
	if bpr > 0 {
		t.Logf("approx disk/row: %.2f bytes", bpr)
		t.Logf("extrapolate 1e8 rows ≈ %s; 1e9 rows ≈ %s",
			humanBytes(int64(bpr*1e8)), humanBytes(int64(bpr*1e9)))
	}
	t.Logf("======================================================")

	require.Equal(t, 0, failBatches)
	require.EqualValues(t, n, totalRows)
}

func buildTempOnlyBatch(stable, child, deviceID string, from, count int, base time.Time) string {
	var sb strings.Builder
	sb.Grow(count * 48)
	sb.WriteString("INSERT INTO ")
	sb.WriteString(child)
	sb.WriteString(" USING ")
	sb.WriteString(stable)
	sb.WriteString(fmt.Sprintf(" TAGS('%s') ", deviceID))
	sb.WriteString("(create_time_,temperature_) VALUES ")
	for i := 0; i < count; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		idx := from + i
		ts := base.Add(time.Duration(idx) * time.Millisecond)
		// 模拟温度 15~35 缓慢变化
		temp := 20.0 + float64(idx%2000)*0.01
		sb.WriteString(fmt.Sprintf("('%s',%g)", ts.Format("2006-01-02 15:04:05.000"), temp))
	}
	return sb.String()
}

// buildBoardcollectorBatch 单设备多行；from 为全局行号起点，时间 base+from*ms 起递增。
func buildBoardcollectorBatch(stable, child, deviceID string, from, count int, base time.Time) string {
	var sb strings.Builder
	sb.Grow(count * 120)
	sb.WriteString("INSERT INTO ")
	sb.WriteString(child)
	sb.WriteString(" USING ")
	sb.WriteString(stable)
	sb.WriteString(fmt.Sprintf(" TAGS('%s') ", deviceID))
	sb.WriteString("(create_time_,latitude_,longitude_,velocity_,pitch_,yaw_,roll_,height_,millisecond_) VALUES ")
	for i := 0; i < count; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		idx := from + i
		ts := base.Add(time.Duration(idx) * time.Millisecond)
		// 模拟无人机轨迹：经纬度缓慢变化
		lat := 31.2000000 + float64(idx%100000)*0.000001
		lng := 121.5000000 + float64(idx%100000)*0.000001
		vel := float32(10 + idx%50)
		pitch := float32((idx % 40) - 20)
		yaw := float32(idx % 360)
		roll := float32((idx % 20) - 10)
		height := float32(50 + idx%200)
		ms := ts.UnixMilli()
		sb.WriteString(fmt.Sprintf("('%s',%.7f,%.7f,%g,%g,%g,%g,%g,%d)",
			ts.Format("2006-01-02 15:04:05.000"), lat, lng, vel, pitch, yaw, roll, height, ms))
	}
	return sb.String()
}

func envInt(k string) int {
	v := os.Getenv(k)
	if v == "" {
		return 0
	}
	x, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return x
}

func mustTaosOK(t *testing.T, sql string) {
	t.Helper()
	require.NoError(t, taosExecOK(sql), sql)
}

func taosExecOK(sql string) error {
	body, err := taosSQL(sql)
	if err != nil {
		return err
	}
	var m struct {
		Code int    `json:"code"`
		Desc string `json:"desc"`
	}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return fmt.Errorf("parse %s: %w", body, err)
	}
	if m.Code != 0 {
		return fmt.Errorf("taos code=%d desc=%s body=%s", m.Code, m.Desc, trimBody(body))
	}
	return nil
}

func trimBody(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}

func compactJSON(s string) string {
	var buf bytes.Buffer
	if json.Compact(&buf, []byte(s)) != nil {
		return trimBody(s)
	}
	return trimBody(buf.String())
}

func dockerDu(path string) (string, error) {
	name := os.Getenv("TDENGINE_DOCKER")
	if name == "" {
		name = "tender_gates"
	}
	out, err := exec.Command("docker", "exec", name, "du", "-sb", path).CombinedOutput()
	if err != nil {
		out2, err2 := exec.Command("docker", "exec", name, "du", "-sh", path).CombinedOutput()
		if err2 != nil {
			return "", fmt.Errorf("du %s: %v %s", path, err, string(out))
		}
		return strings.TrimSpace(string(out2)), nil
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return strings.TrimSpace(string(out)), nil
	}
	b, _ := strconv.ParseInt(fields[0], 10, 64)
	return fmt.Sprintf("%s (%s)", humanBytes(b), fields[0]), nil
}

func parseDuBytes(s string) int64 {
	if i := strings.Index(s, "("); i >= 0 {
		inner := strings.TrimSuffix(strings.TrimSpace(s[i+1:]), ")")
		if n, err := strconv.ParseInt(inner, 10, 64); err == nil {
			return n
		}
	}
	s = strings.TrimSpace(s)
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "K"), strings.HasSuffix(s, "k"):
		mult = 1024
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "M"), strings.HasSuffix(s, "m"):
		mult = 1024 * 1024
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "G"), strings.HasSuffix(s, "g"):
		mult = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(f * float64(mult))
}

func humanBytes(b int64) string {
	if b < 0 {
		return fmt.Sprintf("-%s", humanBytes(-b))
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func taosSQL(sql string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:6041/rest/sql", bytes.NewBufferString(sql))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("root:taosdata")))
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func taosCount(sql string) (int64, error) {
	body, err := taosSQL(sql)
	if err != nil {
		return 0, err
	}
	var m struct {
		Code int             `json:"code"`
		Data [][]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return 0, fmt.Errorf("parse %s: %w", body, err)
	}
	if m.Code != 0 {
		return 0, fmt.Errorf("taos error: %s", body)
	}
	if len(m.Data) == 0 || len(m.Data[0]) == 0 {
		return 0, nil
	}
	switch v := m.Data[0][0].(type) {
	case float64:
		return int64(v), nil
	default:
		return strconv.ParseInt(fmt.Sprint(v), 10, 64)
	}
}

func queryDiskUsage(db string) (string, error) {
	return taosSQL("SELECT db_name,vgroup_id,wal_size,data1,data2,data3,cache_rdb,table_meta,ss,raw_data FROM information_schema.ins_disk_usage WHERE db_name='" + db + "';")
}
