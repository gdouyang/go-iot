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
	Save(product Product, data map[string]interface{}) error
	// 发布模型
	PublishModel(product Product, model tsl.TslData) error
	// 查询属性
	QueryProperty(product Product) (map[string]interface{}, error)
}

// mock
type MockTimeSeries struct {
}

func (t *MockTimeSeries) Save(product Product, data map[string]interface{}) error {
	logs.Info("save timeseries data: ", data)
	return nil
}

func (t *MockTimeSeries) PublishModel(product Product, model tsl.TslData) error {
	logs.Info("PublishModel: ", model)
	return nil
}
func (t *MockTimeSeries) QueryProperty(product Product) (map[string]interface{}, error) {
	logs.Info("QueryProperty: ")
	return nil, nil
}
