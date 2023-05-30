package core

import (
	"go-iot/pkg/core/es"
	"go-iot/pkg/core/tsl"

	"github.com/beego/beego/v2/core/logs"
)

const (
	TIME_SERISE_ES   = "es"   // 时序数据存储策略es
	TIME_SERISE_MOCK = "mock" // 时序数据存储策略mock
)

var timeSeriseMap map[string]TimeSeriesSave = map[string]TimeSeriesSave{}

func init() {
	timeSeriseMap[TIME_SERISE_MOCK] = &MockTimeSeries{}
}

// 获取时序
func GetTimeSeries(id string) TimeSeriesSave {
	return timeSeriseMap[id]
}

// 时序保存
type TimeSeriesSave interface {
	// 发布模型
	PublishModel(product *Product, model tsl.TslData) error
	// 查询属性
	QueryProperty(product *Product, param TimeDataSearchRequest) (map[string]interface{}, error)
	// 保存时序数据
	SaveProperties(product *Product, data map[string]interface{}) error
	SaveEvents(product *Product, eventId string, data map[string]interface{}) error
	SaveLogs(product *Product, data LogData) error
	Del(product *Product) error
}

type LogData struct {
	Type       string `json:"type"`
	DeviceId   string `json:"deviceId"`
	Content    string `json:"content"`
	CreateTime string `json:"createTime"`
}

type TimeDataSearchRequest struct {
	Type        string          `json:"type"`
	DeviceId    string          `json:"deviceId"`
	PageNum     int             `json:"pageNum"`
	PageSize    int             `json:"pageSize"`
	Condition   []es.SearchTerm `json:"condition"`
	SearchAfter []string
}

func (page *TimeDataSearchRequest) PageOffset() int {
	return (page.PageNum - 1) * page.PageSize
}

// mock
type MockTimeSeries struct {
}

func (t *MockTimeSeries) PublishModel(product *Product, model tsl.TslData) error {
	logs.Info("Mock PublishModel: ", model)
	return nil
}
func (t *MockTimeSeries) QueryProperty(product *Product, param TimeDataSearchRequest) (map[string]interface{}, error) {
	logs.Info("Mock QueryProperty: ")
	return nil, nil
}
func (t *MockTimeSeries) SaveProperties(product *Product, data map[string]interface{}) error {
	logs.Info("Mock SaveProperties data: ", data)
	return nil
}
func (t *MockTimeSeries) SaveEvents(product *Product, eventId string, data map[string]interface{}) error {
	logs.Info("Mock SaveEvents data: ", data)
	return nil
}
func (t *MockTimeSeries) SaveLogs(product *Product, data LogData) error {
	logs.Info("Mock SaveLogs data: ", data)
	return nil
}
func (t *MockTimeSeries) Del(product *Product) error {
	logs.Info("Mock Del data: ", product.Id)
	return nil
}
