package codec

import (
	"go-iot/codec/tsl"

	"github.com/beego/beego/v2/core/logs"
)

var timeSeriseMap map[string]TimeSeriesSave = map[string]TimeSeriesSave{}

func init() {
	timeSeriseMap["es"] = &EsTimeSeries{}
	timeSeriseMap["mock"] = &MockTimeSeries{}
}

// 获取时序
func GetTimeSeries(id string) TimeSeriesSave {
	return timeSeriseMap[id]
}

// 时序保存
type TimeSeriesSave interface {
	// 保存时序数据
	Save(product Product, data map[string]interface{})
	// 发布模型
	PublishModel(product Product, model tsl.TslData)
}

// mock
type MockTimeSeries struct {
}

func (t *MockTimeSeries) Save(product Product, data map[string]interface{}) {
	logs.Info("save timeseries data: ", data)
}

func (t *MockTimeSeries) PublishModel(product Product, model tsl.TslData) {
	logs.Info("PublishModel: ", model)
}
