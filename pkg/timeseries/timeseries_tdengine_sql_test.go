package timeseries

import (
	"strings"
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/tsl"

	"github.com/stretchr/testify/require"
)

func TestTdengineId(t *testing.T) {
	var ts TdengineTimeSeries
	require.Equal(t, core.TIME_SERISE_TDENGINE, ts.Id())
}

func TestTdengineColumnNameRewrite(t *testing.T) {
	var ts TdengineTimeSeries
	require.Equal(t, "create_time_", ts.columnNameRewrite("createTime"))
	require.Equal(t, "device_id_", ts.columnNameRewrite("deviceId"))
	require.Equal(t, "obj_0_name_", ts.columnNameRewrite("obj.name"))
	require.Equal(t, "create_time_ TIMESTAMP", ts.columnNameRewrite("createTime", "TIMESTAMP"))
}

func TestTdengineWhereValueRewrite(t *testing.T) {
	var ts TdengineTimeSeries
	require.Equal(t, "'abc'", ts.whereValueRewrite("abc"))
	require.Equal(t, `'a\'b'`, ts.whereValueRewrite("a'b"))
	require.Equal(t, "12", ts.whereValueRewrite(12))
	require.Equal(t, "1.5", ts.whereValueRewrite(1.5))
}

func TestTdengineInsertSql(t *testing.T) {
	var ts TdengineTimeSeries
	sql := ts.insertSql(
		"goiot.properties_p1",
		core.TIME_TYPE_PROP,
		[]string{"current", "light"},
		map[string]any{"deviceId": "d1", "current": 1.2, "light": 3},
		"2026-08-12 10:00:00.000",
	)
	require.Contains(t, sql, "INSERT INTO goiot.properties_d1")
	require.Contains(t, sql, "USING goiot.properties_p1")
	require.Contains(t, sql, "TAGS('d1')")
	require.Contains(t, sql, "create_time_")
	require.Contains(t, sql, "current_")
	require.Contains(t, sql, "light_")
	require.Contains(t, sql, "'2026-08-12 10:00:00.000'")

	// 事件子表带 stable 名，避免同设备不同事件冲突
	esql := ts.insertSql(
		"goiot.event_p1_fire_alarm",
		core.TIME_TYPE_EVENT,
		[]string{"lnt"},
		map[string]any{"deviceId": "d1", "lnt": 1.0},
		"2026-08-12 10:00:00.000",
	)
	require.Contains(t, esql, "INSERT INTO goiot.event_p1_fire_alarm_d1")
	require.Contains(t, esql, "USING goiot.event_p1_fire_alarm")
}

func TestBuildMultiInsertSQL(t *testing.T) {
	var ts TdengineTimeSeries
	f1 := ts.insertFragment("goiot.properties_p1", core.TIME_TYPE_PROP, []string{"light"},
		map[string]any{"deviceId": "d1", "light": 1}, "2026-08-12 10:00:00.000")
	f2 := ts.insertFragment("goiot.properties_p1", core.TIME_TYPE_PROP, []string{"light"},
		map[string]any{"deviceId": "d2", "light": 2}, "2026-08-12 10:00:00.001")
	sql := buildMultiInsertSQL([]string{f1, f2})
	require.True(t, strings.HasPrefix(sql, "INSERT INTO "))
	require.Contains(t, sql, f1)
	require.Contains(t, sql, ", ")
	require.Contains(t, sql, f2)
	require.True(t, strings.HasSuffix(sql, ";"))
}

func TestNextTdCreateTimeMonotonic(t *testing.T) {
	a := nextTdCreateTime()
	b := nextTdCreateTime()
	require.NotEqual(t, a, b)
}

func TestTdengineWhereBuilders(t *testing.T) {
	var ts TdengineTimeSeries
	cases := []struct {
		name string
		term core.SearchTerm
		want string
	}{
		{"eq", core.SearchTerm{Key: "light", Value: 1, Oper: core.EQ}, "light_ = 1"},
		{"gt", core.SearchTerm{Key: "light", Value: 1, Oper: core.GT}, "light_ > 1"},
		{"gte", core.SearchTerm{Key: "light", Value: 1, Oper: core.GTE}, "light_ >= 1"},
		{"lt", core.SearchTerm{Key: "light", Value: 9, Oper: core.LT}, "light_ < 9"},
		{"lte", core.SearchTerm{Key: "light", Value: 9, Oper: core.LTE}, "light_ <= 9"},
		{"neq", core.SearchTerm{Key: "light", Value: 0, Oper: core.NEQ}, "light_ != 0"},
		{"like", core.SearchTerm{Key: "type", Value: "%on%", Oper: core.LIKE}, "type_ LIKE '%on%'"},
		{"in", core.SearchTerm{Key: "light", Value: []any{1, 2}, Oper: core.IN}, "light_ IN ( 1,2)"},
		{"btw", core.SearchTerm{Key: "light", Value: []any{1, 5}, Oper: core.BTW}, "light_ >= 1 AND light_ <= 5"},
		{"notnull", core.SearchTerm{Key: "light", Value: true, Oper: core.NOTNULL}, "light_ is not null "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sb := strings.Builder{}
			ts.where(&sb, []core.SearchTerm{tc.term})
			require.Contains(t, sb.String(), tc.want)
		})
	}
}

func TestTdengineStableNames(t *testing.T) {
	var ts TdengineTimeSeries
	p, err := core.NewProduct("prod-a", map[string]string{}, core.TIME_SERISE_TDENGINE, `{"properties":[{"id":"x","name":"x","type":"int"}],"events":[],"functions":[]}`)
	require.NoError(t, err)
	require.Equal(t, "goiot.properties_prod_a", ts.getStableName(p, core.TIME_TYPE_PROP))
	require.Equal(t, "goiot.devicelogs_prod_a", ts.getStableName(p, core.TIME_TYPE_LOGS))
	require.Equal(t, "goiot.event_prod_a_fire_alarm", ts.getEventStableName(p, core.TIME_TYPE_EVENT, "fire-alarm"))

	// 大写产品 / 事件：折叠为小写，与全小写 ID 共用表名
	p2, err := core.NewProduct("MyProduct", map[string]string{}, core.TIME_SERISE_TDENGINE, `{"properties":[{"id":"x","name":"x","type":"int"}],"events":[],"functions":[]}`)
	require.NoError(t, err)
	require.Equal(t, "goiot.properties_myproduct", ts.getStableName(p2, core.TIME_TYPE_PROP))
	require.Equal(t, "goiot.event_myproduct_firealarm", ts.getEventStableName(p2, core.TIME_TYPE_EVENT, "FireAlarm"))

	p3, err := core.NewProduct("myproduct", map[string]string{}, core.TIME_SERISE_TDENGINE, `{"properties":[{"id":"x","name":"x","type":"int"}],"events":[],"functions":[]}`)
	require.NoError(t, err)
	require.Equal(t, ts.getStableName(p2, core.TIME_TYPE_PROP), ts.getStableName(p3, core.TIME_TYPE_PROP))
}

func TestTdNamePart(t *testing.T) {
	require.Equal(t, "boardcollector", tdNamePart("boardcollector"))
	require.Equal(t, "myproduct", tdNamePart("MyProduct"))
	require.Equal(t, "myproduct", tdNamePart("myproduct"))
	require.Equal(t, "myproduct", tdNamePart("MYPRODUCT"))
	// '-' → '_'
	require.Equal(t, "prod_a", tdNamePart("prod-a"))
	require.Equal(t, "my_product", tdNamePart("my_product"))
}

func TestTdengineCreateSqlColumnObject(t *testing.T) {
	var ts TdengineTimeSeries
	obj := &tsl.PropertyObject{}
	// 通过 JSON 构建 object 属性更稳妥
	d := tsl.NewTslData()
	err := d.FromJson(`{
	  "properties":[{
	    "id":"obj","name":"obj","type":"object",
	    "properties":[
	      {"id":"name","name":"n","type":"string"},
	      {"id":"v","name":"v","type":"int"}
	    ]
	  }],
	  "events":[],"functions":[]
	}`)
	require.NoError(t, err)
	p := d.Properties[0]
	sb := strings.Builder{}
	ts.createSqlColumn(&sb, p.GetId(), p)
	sql := sb.String()
	require.Contains(t, sql, "obj_0_name_")
	require.Contains(t, sql, "NCHAR(32)")
	require.Contains(t, sql, "obj_0_v_")
	require.Contains(t, sql, "INT")
	_ = obj
}
