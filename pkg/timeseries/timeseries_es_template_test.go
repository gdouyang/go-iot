package timeseries

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/es"
	logs "go-iot/pkg/logger"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/stretchr/testify/require"
)

// 集成测试：连本地 ES，验证 PublishModel 模板名/pattern 与月索引创建。
// 默认 http://localhost:9200；可用 ES_URL 覆盖。ES 不可达时跳过。
func TestEsPublishModelTemplateAndIndexLive(t *testing.T) {
	esURL := strings.TrimSpace(os.Getenv("ES_URL"))
	if esURL == "" {
		esURL = "http://localhost:9200"
	}
	if !esReachable(t, esURL) {
		t.Skipf("ES not reachable at %s", esURL)
	}

	logs.InitNop()
	es.DefaultEsConfig.Url = esURL
	es.DefaultEsConfig.NumberOfShards = "1"
	es.DefaultEsConfig.NumberOfReplicas = "0"

	productID := "MyProduct_IT"
	eventID := "FireAlarm"
	// 索引段：仅转小写
	pidEnc := esIndexPart(productID)
	eidEnc := esIndexPart(eventID)
	require.Equal(t, "myproduct_it", pidEnc)
	require.Equal(t, "firealarm", eidEnc)

	// 清理上次残留
	cleanupTemplates(t,
		fmt.Sprintf("%s-%s-template", properties_const, pidEnc),
		fmt.Sprintf("%s-%s-%s-template", event_const, pidEnc, eidEnc),
		fmt.Sprintf("%s-%s-template", devicelogs_const, pidEnc),
	)
	cleanupIndices(t,
		fmt.Sprintf("%s-%s-*", properties_const, pidEnc),
		fmt.Sprintf("%s-%s-%s-*", event_const, pidEnc, eidEnc),
		fmt.Sprintf("%s-%s-*", devicelogs_const, pidEnc),
	)
	t.Cleanup(func() {
		cleanupTemplates(t,
			fmt.Sprintf("%s-%s-template", properties_const, pidEnc),
			fmt.Sprintf("%s-%s-%s-template", event_const, pidEnc, eidEnc),
			fmt.Sprintf("%s-%s-template", devicelogs_const, pidEnc),
		)
		cleanupIndices(t,
			fmt.Sprintf("%s-%s-*", properties_const, pidEnc),
			fmt.Sprintf("%s-%s-%s-*", event_const, pidEnc, eidEnc),
			fmt.Sprintf("%s-%s-*", devicelogs_const, pidEnc),
		)
	})

	meta := `{"properties":[{"id":"temp","name":"温度","type":"float"},{"id":"online","name":"在线","type":"bool"}],"events":[{"id":"FireAlarm","name":"火警","type":"object","properties":[{"id":"level","name":"等级","type":"int"}]}],"functions":[]}`
	product, err := core.NewProduct(productID, map[string]string{}, core.TIME_SERISE_ES, meta)
	require.NoError(t, err)

	var ts EsTimeSeries
	// 1) 期望索引名（业务 ID 转小写）
	now := time.Date(2026, 8, 12, 10, 0, 0, 0, time.Local)
	propsMonth := ts.getMonthIndex(product, properties_const, now)
	eventMonth := ts.getMonthEventIndex(product, event_const, eventID, now)
	logsMonth := ts.getMonthIndex(product, devicelogs_const, now)
	require.Equal(t, "goiot-properties-myproduct_it-202608", propsMonth)
	require.Equal(t, "goiot-event-myproduct_it-firealarm-202608", eventMonth)
	require.Equal(t, "goiot-devicelogs-myproduct_it-202608", logsMonth)

	// 2) PublishModel 创建 legacy index template
	model := product.GetTsl()
	require.NotNil(t, model)
	require.NoError(t, ts.PublishModel(product, *model))

	// 3) 校验模板存在且 pattern 正确
	type tplExpect struct {
		name    string
		pattern string
	}
	expects := []tplExpect{
		{
			name:    fmt.Sprintf("%s-%s-template", properties_const, pidEnc),
			pattern: fmt.Sprintf("%s-%s-*", properties_const, pidEnc),
		},
		{
			name:    fmt.Sprintf("%s-%s-%s-template", event_const, pidEnc, eidEnc),
			pattern: fmt.Sprintf("%s-%s-%s-*", event_const, pidEnc, eidEnc),
		},
		{
			name:    fmt.Sprintf("%s-%s-template", devicelogs_const, pidEnc),
			pattern: fmt.Sprintf("%s-%s-*", devicelogs_const, pidEnc),
		},
	}
	for _, e := range expects {
		body := getTemplate(t, e.name)
		require.Contains(t, body, e.name, "template %s missing in response", e.name)
		// legacy GET template 返回 { name: { index_patterns: [...] } }
		patterns := gjsonArrayStrings(body, e.name+".index_patterns")
		require.Contains(t, patterns, e.pattern, "template %s patterns=%v", e.name, patterns)
		t.Logf("template OK: %s patterns=%v", e.name, patterns)
	}

	// 4) 写入文档触发月索引创建，验证 mapping 来自模板（createTime=date, deviceId=keyword）
	// CreateDoc 在 ES8 上必须带 DocumentID（空 ID 会走 /_create/ 导致 405）
	require.NoError(t, es.CreateDoc(propsMonth, "doc-props-1", map[string]any{
		"deviceId":   "dev-1",
		"temp":       36.5,
		"online":     true,
		"createTime": time.Now().Format("2006-01-02 15:04:05.000"),
	}))
	require.NoError(t, es.CreateDoc(eventMonth, "doc-event-1", map[string]any{
		"deviceId":   "dev-1",
		"level":      2,
		"createTime": time.Now().Format("2006-01-02 15:04:05.000"),
	}))
	require.NoError(t, es.CreateDoc(logsMonth, "doc-logs-1", map[string]any{
		"deviceId":   "dev-1",
		"type":       "info",
		"content":    "hello",
		"createTime": time.Now().Format("2006-01-02 15:04:05.000"),
	}))

	// 索引名应全小写
	for _, idx := range []string{propsMonth, eventMonth, logsMonth} {
		require.Equal(t, strings.ToLower(idx), idx, "index must be lowercase: %s", idx)
		mapping := getIndexMapping(t, idx)
		require.Contains(t, mapping, idx)
		// createTime 应为 date
		ctType := gjsonString(mapping, idx+".mappings.properties.createTime.type")
		require.Equal(t, "date", ctType, "index %s createTime type", idx)
		devType := gjsonString(mapping, idx+".mappings.properties.deviceId.type")
		require.Equal(t, "keyword", devType, "index %s deviceId type", idx)
		t.Logf("index OK: %s createTime=%s deviceId=%s", idx, ctType, devType)
	}

	// properties 特有字段（bool 在 createElasticProperty 中映射为 keyword）
	propsMap := getIndexMapping(t, propsMonth)
	require.Equal(t, "float", gjsonString(propsMap, propsMonth+".mappings.properties.temp.type"))
	require.Equal(t, "keyword", gjsonString(propsMap, propsMonth+".mappings.properties.online.type"))

	// 5) 大小写折叠后索引段相同
	require.Equal(t, esIndexPart(productID), esIndexPart("myproduct_it"))
	require.Equal(t, ts.getIndex(product, properties_const),
		properties_const+"-"+esIndexPart("myproduct_it"))
}

func esReachable(t *testing.T, url string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func getTemplate(t *testing.T, name string) string {
	t.Helper()
	req := esapi.IndicesGetTemplateRequest{Name: []string{name}}
	resp, err := es.DoRequest(req)
	require.NoError(t, err)
	require.False(t, resp.IsError, "get template %s: %s", name, resp.Data)
	return resp.Data
}

func getIndexMapping(t *testing.T, index string) string {
	t.Helper()
	req := esapi.IndicesGetMappingRequest{Index: []string{index}}
	resp, err := es.DoRequest(req)
	require.NoError(t, err)
	require.False(t, resp.IsError, "get mapping %s: %s", index, resp.Data)
	return resp.Data
}

func cleanupTemplates(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		req := esapi.IndicesDeleteTemplateRequest{Name: name}
		resp, err := es.DoRequest(req)
		if err != nil {
			t.Logf("delete template %s err: %v", name, err)
			continue
		}
		if resp.IsError && !resp.Is404() {
			t.Logf("delete template %s: %s", name, resp.Data)
		}
	}
}

func cleanupIndices(t *testing.T, patterns ...string) {
	t.Helper()
	// ES8 默认禁止通配删除，先 cat 再按名删除
	var names []string
	for _, p := range patterns {
		list, err := es.CatIndices(p)
		if err != nil {
			t.Logf("cat indices %s: %v", p, err)
			continue
		}
		names = append(names, list...)
	}
	if len(names) == 0 {
		return
	}
	if err := es.DeleteIndex(names...); err != nil {
		t.Logf("delete indices %v: %v", names, err)
	}
}

// 轻量 JSON 取值，避免再引 gjson 到测试（es 包已用 gjson，这里直接手写简单路径）
func gjsonString(jsonStr, path string) string {
	var root any
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return ""
	}
	cur := root
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = m[part]
		if !ok {
			return ""
		}
	}
	switch v := cur.(type) {
	case string:
		return v
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func gjsonArrayStrings(jsonStr, path string) []string {
	var root any
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil
	}
	cur := root
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[part]
		if !ok {
			return nil
		}
	}
	arr, ok := cur.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

