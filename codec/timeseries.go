package codec

import "github.com/beego/beego/v2/core/logs"

var timeSeriseMap map[string]TimeSeries = map[string]TimeSeries{}

func init() {
	timeSeriseMap["es"] = &EsTimeSeries{}
	timeSeriseMap["mock"] = &MockTimeSeries{}
}

// 获取时序
func GetTimeSeries(id string) TimeSeries {
	return timeSeriseMap[id]
}

// es时序保存
type EsTimeSeries struct {
}

func (t *EsTimeSeries) Save(productId string, data map[string]interface{}) {
}

// mock
type MockTimeSeries struct {
}

func (t *MockTimeSeries) Save(productId string, data map[string]interface{}) {
	logs.Info("save timeseries data: ", data)
}
