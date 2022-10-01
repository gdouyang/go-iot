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
	Save(productId string, data map[string]interface{})
}

type TimeSeriesModel interface {
	// 发布模型
	PublishModel(product string, model tsl.TslData)
}

// mock
type MockTimeSeries struct {
}

func (t *MockTimeSeries) Save(productId string, data map[string]interface{}) {
	logs.Info("save timeseries data: ", data)
}

func (t *MockTimeSeries) PublishModel(product string, model tsl.TslData) {
	logs.Info("PublishModel: ", model)
}
