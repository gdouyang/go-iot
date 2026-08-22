package timeseries

import (
	"go-iot/pkg/core"
	"go-iot/pkg/tsl"
)

func init() {
	core.RegisterTimeSeries(&NoopTimeSeries{})
}

// NoopTimeSeries 空操作时序存储实现，专用于大规模高并发压测/基准测试场景。
// 零锁争用、零内存分配、零 I/O 阻塞、零日志打印。
type NoopTimeSeries struct{}

func (t *NoopTimeSeries) Id() string {
	return core.TIME_SERISE_NOOP
}

func (t *NoopTimeSeries) PublishModel(product *core.Product, model tsl.TslData) error {
	return nil
}

func (t *NoopTimeSeries) QueryProperty(product *core.Product, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	return map[string]interface{}{"total": 0, "data": []any{}}, nil
}

func (t *NoopTimeSeries) QueryLogs(product *core.Product, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	return map[string]interface{}{"total": 0, "data": []any{}}, nil
}

func (t *NoopTimeSeries) QueryEvent(product *core.Product, eventId string, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	return map[string]interface{}{"total": 0, "data": []any{}}, nil
}

func (t *NoopTimeSeries) SaveProperties(product *core.Product, data map[string]interface{}) error {
	return nil
}

func (t *NoopTimeSeries) SaveEvents(product *core.Product, eventId string, data map[string]interface{}) error {
	return nil
}

func (t *NoopTimeSeries) SaveLogs(product *core.Product, data core.LogData) error {
	return nil
}

func (t *NoopTimeSeries) Del(product *core.Product) error {
	return nil
}
