package timeseries

import (
	"testing"
	"time"

	"go-iot/pkg/core"

	"github.com/stretchr/testify/require"
)

func TestEsIndexPartLowercase(t *testing.T) {
	require.Equal(t, "boardcollector", esIndexPart("boardcollector"))
	require.Equal(t, "abc-def", esIndexPart("abc-def"))
	require.Equal(t, "boardcollector", esIndexPart("BoardCollector"))
	require.Equal(t, "myproduct", esIndexPart("MyProduct"))
	require.Equal(t, "myproduct", esIndexPart("MYPRODUCT"))
	require.Equal(t, "myproduct", esIndexPart("myproduct"))
	require.Equal(t, "my_product", esIndexPart("my_product"))
	require.Equal(t, "abc", esIndexPart("ABC"))
	// 去空白
	require.Equal(t, "dev-01", esIndexPart("  Dev-01  "))

	// ES 保留 '-'；TD 把 '-' 换成 '_'
	require.Equal(t, "prod-a", esIndexPart("prod-a"))
	require.Equal(t, "prod_a", tdNamePart("prod-a"))
	require.Equal(t, "myproduct", tdNamePart("MyProduct"))
}

func TestEsIndexNamesWithUppercaseProduct(t *testing.T) {
	var ts EsTimeSeries
	p, err := core.NewProduct("MyProduct", map[string]string{}, core.TIME_SERISE_ES,
		`{"properties":[{"id":"t","name":"t","type":"int"}],"events":[{"id":"FireAlarm","name":"f","type":"string"}],"functions":[]}`)
	require.NoError(t, err)

	require.Equal(t, "goiot-properties-myproduct", ts.getIndex(p, properties_const))
	require.Equal(t, "goiot-properties-myproduct-202608",
		ts.getMonthIndex(p, properties_const, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)))
	require.Equal(t, "goiot-event-myproduct-firealarm", ts.getEventIndex(p, event_const, "FireAlarm"))
	require.Equal(t, "goiot-devicelogs-myproduct", ts.getIndex(p, devicelogs_const))

	// 全小写业务 ID 与大写折叠后同一索引段
	p2, err := core.NewProduct("myproduct", map[string]string{}, core.TIME_SERISE_ES,
		`{"properties":[{"id":"t","name":"t","type":"int"}],"events":[],"functions":[]}`)
	require.NoError(t, err)
	require.Equal(t, ts.getIndex(p, properties_const), ts.getIndex(p2, properties_const))

	// 保留策略用原始产品 ID 匹配
	meta, ok := MatchProductTimeseriesIndex("goiot-properties-myproduct-202601", "MyProduct")
	require.True(t, ok)
	require.Equal(t, "myproduct", meta.ProductID)
}
