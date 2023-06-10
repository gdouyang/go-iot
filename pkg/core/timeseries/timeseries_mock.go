package core

import (
	"go-iot/pkg/core"
	"go-iot/pkg/core/tsl"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	core.RegisterTimeSeries(&MockTimeSeries{})
}

// mock
type MockTimeSeries struct {
}

func (t *MockTimeSeries) Id() string {
	return core.TIME_SERISE_MOCK
}
func (t *MockTimeSeries) PublishModel(product *core.Product, model tsl.TslData) error {
	logs.Info("Mock PublishModel: ", model)
	return nil
}
func (t *MockTimeSeries) QueryProperty(product *core.Product, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	logs.Info("Mock QueryProperty: ")
	return nil, nil
}

func (t *MockTimeSeries) QueryLogs(product *core.Product, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	logs.Info("Mock QueryLogs: ")
	return nil, nil
}
func (t *MockTimeSeries) QueryEvent(product *core.Product, eventId string, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	logs.Info("Mock QueryEvent: ")
	return nil, nil
}
func (t *MockTimeSeries) SaveProperties(product *core.Product, data map[string]interface{}) error {
	logs.Info("Mock SaveProperties data: ", data)
	return nil
}
func (t *MockTimeSeries) SaveEvents(product *core.Product, eventId string, data map[string]interface{}) error {
	logs.Info("Mock SaveEvents data: ", data)
	return nil
}
func (t *MockTimeSeries) SaveLogs(product *core.Product, data core.LogData) error {
	logs.Info("Mock SaveLogs data: ", data)
	return nil
}
func (t *MockTimeSeries) Del(product *core.Product) error {
	logs.Info("Mock Del data: ", product.Id)
	return nil
}
