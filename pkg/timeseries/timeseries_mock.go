package timeseries

import (
	"go-iot/pkg/core"
	"go-iot/pkg/tsl"

	logs "go-iot/pkg/logger"
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
	logs.Infof("Mock PublishModel: %v", model)
	return nil
}
func (t *MockTimeSeries) QueryProperty(product *core.Product, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	logs.Infof("Mock QueryProperty: ")
	return nil, nil
}

func (t *MockTimeSeries) QueryLogs(product *core.Product, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	logs.Infof("Mock QueryLogs: ")
	return nil, nil
}
func (t *MockTimeSeries) QueryEvent(product *core.Product, eventId string, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	logs.Infof("Mock QueryEvent: ")
	return nil, nil
}
func (t *MockTimeSeries) SaveProperties(product *core.Product, data map[string]interface{}) error {
	logs.Infof("Mock SaveProperties data: %v", data)
	return nil
}
func (t *MockTimeSeries) SaveEvents(product *core.Product, eventId string, data map[string]interface{}) error {
	logs.Infof("Mock SaveEvents data: %v", data)
	return nil
}
func (t *MockTimeSeries) SaveLogs(product *core.Product, data core.LogData) error {
	logs.Infof("Mock SaveLogs data: %v", data)
	return nil
}
func (t *MockTimeSeries) Del(product *core.Product) error {
	logs.Infof("Mock Del data: %s", product.Id)
	return nil
}
