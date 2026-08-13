package timeseries

import (
	"os"
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/tsl"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

func TestNormalizeTdType(t *testing.T) {
	require.Equal(t, "INT", normalizeTdType("INT"))
	require.Equal(t, "NCHAR", normalizeTdType("NCHAR(32)"))
	require.Equal(t, "NCHAR", normalizeTdType("nchar"))
	require.Equal(t, "DOUBLE", normalizeTdType("DOUBLE"))
	require.True(t, tdTypesCompatible("NCHAR(64)", "NCHAR(32)"))
	require.False(t, tdTypesCompatible("INT", "DOUBLE"))
}

func TestExpandPropertyColumns(t *testing.T) {
	var ts TdengineTimeSeries
	d := tsl.NewTslData()
	require.NoError(t, d.FromJson(`{
	  "properties":[
	    {"id":"light","name":"l","type":"int"},
	    {"id":"obj","name":"o","type":"object","properties":[
	      {"id":"name","name":"n","type":"string"},
	      {"id":"v","name":"v","type":"double"}
	    ]}
	  ],
	  "events":[],"functions":[]
	}`))
	var cols []tdColDef
	for _, p := range d.Properties {
		cols = append(cols, ts.expandPropertyColumns(p.GetId(), p)...)
	}
	names := map[string]string{}
	for _, c := range cols {
		names[c.Name] = c.Type
	}
	require.Equal(t, "INT", names["light_"])
	require.Equal(t, "NCHAR", names["obj_0_name_"])
	require.Equal(t, "DOUBLE", names["obj_0_v_"])
}

func TestTdengineSchemaEvolve(t *testing.T) {
	logs.InitNop()
	if os.Getenv("TDENGINE_TEST") != "1" {
		t.Skip("set TDENGINE_TEST=1")
	}
	var ts TdengineTimeSeries
	// 独立库名避免污染：仍写 goiot，用独立 productId
	pid := "schema_evo_test"
	p, err := core.NewProduct(pid, map[string]string{}, core.TIME_SERISE_TDENGINE, `{"properties":[{"id":"a","name":"a","type":"int"}],"events":[],"functions":[]}`)
	require.NoError(t, err)
	d1 := tsl.NewTslData()
	require.NoError(t, d1.FromJson(`{"properties":[{"id":"a","name":"a","type":"int"}],"events":[{"id":"e1","name":"e1","type":"string"}],"functions":[]}`))
	p.TslData = d1
	_ = ts.Del(p)

	require.NoError(t, ts.PublishModel(p, *d1))
	cols, err := ts.describeColumns(ts.getStableName(p, core.TIME_TYPE_PROP))
	require.NoError(t, err)
	require.Contains(t, cols, "a_")
	require.NotContains(t, cols, "b_")

	// 增列 + 新事件
	d2 := tsl.NewTslData()
	require.NoError(t, d2.FromJson(`{"properties":[
	  {"id":"a","name":"a","type":"int"},
	  {"id":"b","name":"b","type":"double"}
	],"events":[
	  {"id":"e1","name":"e1","type":"string"},
	  {"id":"e2","name":"e2","type":"int"}
	],"functions":[]}`))
	require.NoError(t, ts.PublishModel(p, *d2))
	cols, err = ts.describeColumns(ts.getStableName(p, core.TIME_TYPE_PROP))
	require.NoError(t, err)
	require.Contains(t, cols, "a_")
	require.Contains(t, cols, "b_")
	exist, err := ts.stableExists(ts.getEventStableName(p, core.TIME_TYPE_EVENT, "e2"))
	require.NoError(t, err)
	require.True(t, exist)

	// 删列 a、删事件 e1
	d3 := tsl.NewTslData()
	require.NoError(t, d3.FromJson(`{"properties":[
	  {"id":"b","name":"b","type":"double"}
	],"events":[
	  {"id":"e2","name":"e2","type":"int"}
	],"functions":[]}`))
	require.NoError(t, ts.PublishModel(p, *d3))
	cols, err = ts.describeColumns(ts.getStableName(p, core.TIME_TYPE_PROP))
	require.NoError(t, err)
	require.NotContains(t, cols, "a_")
	require.Contains(t, cols, "b_")
	exist, err = ts.stableExists(ts.getEventStableName(p, core.TIME_TYPE_EVENT, "e1"))
	require.NoError(t, err)
	require.False(t, exist, "obsolete event stable should be dropped")

	// 类型变更应失败
	d4 := tsl.NewTslData()
	require.NoError(t, d4.FromJson(`{"properties":[{"id":"b","name":"b","type":"int"}],"events":[],"functions":[]}`))
	err = ts.PublishModel(p, *d4)
	require.Error(t, err)
	require.Contains(t, err.Error(), "type mismatch")

	_ = ts.Del(p)
}
