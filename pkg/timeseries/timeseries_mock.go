package timeseries

import (
	"fmt"
	"sync"

	"go-iot/pkg/core"
	"go-iot/pkg/tsl"

	logs "go-iot/pkg/logger"
)

func init() {
	core.RegisterTimeSeries(&MockTimeSeries{})
}

// mock 写入记录，供单测/集成测断言 SaveProperties 等是否真的被调用。
type mockRecord struct {
	sync.Mutex
	properties []map[string]interface{}
	events     []mockEvent
	logs       []core.LogData
}

type mockEvent struct {
	EventId string
	Data    map[string]interface{}
}

var mockRec mockRecord

// MockReset 清空 mock 写入记录（测试 setup 时调用）。
func MockReset() {
	mockRec.Lock()
	defer mockRec.Unlock()
	mockRec.properties = nil
	mockRec.events = nil
	mockRec.logs = nil
}

// MockLastProperties 最近一次 SaveProperties 的 data（无则 nil）。
func MockLastProperties() map[string]interface{} {
	mockRec.Lock()
	defer mockRec.Unlock()
	if len(mockRec.properties) == 0 {
		return nil
	}
	return mockRec.properties[len(mockRec.properties)-1]
}

// MockPropertiesCount SaveProperties 调用次数。
func MockPropertiesCount() int {
	mockRec.Lock()
	defer mockRec.Unlock()
	return len(mockRec.properties)
}

// MockAllProperties 返回已保存属性的浅拷贝切片（从旧到新）。
func MockAllProperties() []map[string]interface{} {
	mockRec.Lock()
	defer mockRec.Unlock()
	out := make([]map[string]interface{}, len(mockRec.properties))
	for i, p := range mockRec.properties {
		cp := make(map[string]interface{}, len(p))
		for k, v := range p {
			cp[k] = v
		}
		out[i] = cp
	}
	return out
}

// MockLastEvent 最近一次 SaveEvents。
func MockLastEvent() (eventId string, data map[string]interface{}) {
	mockRec.Lock()
	defer mockRec.Unlock()
	if len(mockRec.events) == 0 {
		return "", nil
	}
	e := mockRec.events[len(mockRec.events)-1]
	return e.EventId, e.Data
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
	// 返回最近一次匹配 deviceId 的属性，便于测试 Query 路径
	mockRec.Lock()
	defer mockRec.Unlock()
	for i := len(mockRec.properties) - 1; i >= 0; i-- {
		p := mockRec.properties[i]
		if param.DeviceId == "" || fmtDeviceId(p) == param.DeviceId {
			return map[string]interface{}{
				"total": 1,
				"data":  []map[string]interface{}{p},
			}, nil
		}
	}
	return map[string]interface{}{"total": 0, "data": []any{}}, nil
}

func fmtDeviceId(data map[string]interface{}) string {
	if data == nil {
		return ""
	}
	if v, ok := data["deviceId"]; ok {
		return fmt.Sprint(v)
	}
	return ""
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
	// 拷贝一份，避免后续修改影响断言
	cp := make(map[string]interface{}, len(data))
	for k, v := range data {
		cp[k] = v
	}
	mockRec.Lock()
	mockRec.properties = append(mockRec.properties, cp)
	mockRec.Unlock()
	return nil
}
func (t *MockTimeSeries) SaveEvents(product *core.Product, eventId string, data map[string]interface{}) error {
	logs.Infof("Mock SaveEvents data: %v", data)
	cp := make(map[string]interface{}, len(data))
	for k, v := range data {
		cp[k] = v
	}
	mockRec.Lock()
	mockRec.events = append(mockRec.events, mockEvent{EventId: eventId, Data: cp})
	mockRec.Unlock()
	return nil
}
func (t *MockTimeSeries) SaveLogs(product *core.Product, data core.LogData) error {
	logs.Infof("Mock SaveLogs data: %v", data)
	mockRec.Lock()
	mockRec.logs = append(mockRec.logs, data)
	mockRec.Unlock()
	return nil
}
func (t *MockTimeSeries) Del(product *core.Product) error {
	logs.Infof("Mock Del data: %s", product.Id)
	return nil
}
