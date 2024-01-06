package core

import (
	"go-iot/pkg/tsl"
	"log"
	"sync"
)

const (
	TIME_SERISE_ES       = "es"         // 时序数据存储策略es
	TIME_SERISE_TDENGINE = "tdengine"   // 时序数据存储策略Tdengine
	TIME_SERISE_MOCK     = "mock"       // 时序数据存储策略mock
	TIME_TYPE_PROP       = "properties" // 物模型-属性
	TIME_TYPE_LOGS       = "devicelogs" // 物模型-日志
	TIME_TYPE_EVENT      = "event"      // 物模型-事件
)
const (
	IN      = "IN"
	EQ      = "EQ"      // Equal to
	NEQ     = "NEQ"     // Not Equal to
	GT      = "GT"      // Greater than
	GTE     = "GTE"     // Greater than or Equal
	LT      = "LT"      // less then
	LTE     = "LTE"     // less then or Equal
	LIKE    = "LIKE"    // like
	BTW     = "BTW"     // between
	NOTNULL = "NOTNULL" // not null
)

var timeSeriseMap sync.Map

// 注册时序数据存储策略
func RegisterTimeSeries(ts TimeSeriesSave) {
	log.Printf("Register timeseries [%s]", ts.Id())
	timeSeriseMap.Store(ts.Id(), ts)
}

// 获取时序数据存储策略
func GetTimeSeries(id string) TimeSeriesSave {
	val, _ := timeSeriseMap.Load(id)
	return val.(TimeSeriesSave)
}

// 时序数据存储策略
type TimeSeriesSave interface {
	Id() string
	// 发布模型
	PublishModel(product *Product, model tsl.TslData) error
	// 查询属性
	QueryProperty(product *Product, param TimeDataSearchRequest) (map[string]interface{}, error)
	QueryLogs(product *Product, param TimeDataSearchRequest) (map[string]interface{}, error)
	QueryEvent(product *Product, eventId string, param TimeDataSearchRequest) (map[string]interface{}, error)
	// 保存时序数据
	SaveProperties(product *Product, data map[string]interface{}) error
	SaveEvents(product *Product, eventId string, data map[string]interface{}) error
	SaveLogs(product *Product, data LogData) error
	Del(product *Product) error
}

type LogData struct {
	Type       string `json:"type"`
	TraceId    string `json:"traceId"`
	DeviceId   string `json:"deviceId"`
	Content    string `json:"content"`
	CreateTime string `json:"createTime"`
}

// 时序数据查询结构
type TimeDataSearchRequest struct {
	DeviceId    string       `json:"deviceId"`
	PageNum     int          `json:"pageNum"`
	PageSize    int          `json:"pageSize"`
	Condition   []SearchTerm `json:"condition"`
	SearchAfter []any        `json:"searchAfter"`
}

func (page *TimeDataSearchRequest) PageOffset() int {
	return (page.PageNum - 1) * page.PageSize
}

type SearchTerm struct {
	Key   string `json:"key"`   // 查询的字段
	Value any    `json:"value"` // 值
	Oper  string `json:"oper"`  // 操作符 IN, EQ, NEQ, GT ,LT ,LIKE;默认(EQ)
}
