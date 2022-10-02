package codec

import (
	"go-iot/codec/tsl"

	"github.com/beego/beego/v2/core/logs"
)

const (
	TIME_SERISE_ES   = "es"
	TIME_SERISE_MOCK = "mock"
)

var timeSeriseMap map[string]TimeSeriesSave = map[string]TimeSeriesSave{}

func init() {
	timeSeriseMap[TIME_SERISE_ES] = &EsTimeSeries{}
	timeSeriseMap[TIME_SERISE_MOCK] = &MockTimeSeries{}
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
	PublishModel(product Product, model tsl.TslData) error
}

// mock
type MockTimeSeries struct {
}

func (t *MockTimeSeries) Save(product Product, data map[string]interface{}) {
	logs.Info("save timeseries data: ", data)
}

func (t *MockTimeSeries) PublishModel(product Product, model tsl.TslData) error {
	logs.Info("PublishModel: ", model)
	return nil
}
