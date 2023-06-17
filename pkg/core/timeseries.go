package core

import (
	"go-iot/pkg/core/tsl"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

const (
	TIME_SERISE_ES   = "es"   // 时序数据存储策略es
	TIME_SERISE_MOCK = "mock" // 时序数据存储策略mock
	TIME_TYPE_PROP   = "properties"
	TIME_TYPE_LOGS   = "devicelogs"
	TIME_TYPE_EVENT  = "event"
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

func RegisterTimeSeries(ts TimeSeriesSave) {
	logs.Info("Register timeseries [%s]", ts.Id())
	timeSeriseMap.Store(ts.Id(), ts)
}

// 获取时序
func GetTimeSeries(id string) TimeSeriesSave {
	val, _ := timeSeriseMap.Load(id)
	return val.(TimeSeriesSave)
}

// 时序保存
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
	DeviceId   string `json:"deviceId"`
	Content    string `json:"content"`
	CreateTime string `json:"createTime"`
}

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
	Oper  string `json:"oper"`  // 操作符IN,EQ,GT,LE,LIKE;默认(EQ)
}
