package timeseries

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/core/tsl"
	"go-iot/pkg/es"
	"go-iot/pkg/eventbus"
	"strings"
	"time"

	logs "go-iot/pkg/logger"
)

func init() {
	core.RegisterTimeSeries(&EsTimeSeries{})
}

const (
	properties_const = es.Prefix + core.TIME_TYPE_PROP
	event_const      = es.Prefix + core.TIME_TYPE_EVENT
	devicelogs_const = es.Prefix + core.TIME_TYPE_LOGS
	timeformt        = "2006-01-02 15:04:05.000"
)

// es时序保存
type EsTimeSeries struct {
}

func (t *EsTimeSeries) Id() string {
	return core.TIME_SERISE_ES
}

func (t *EsTimeSeries) PublishModel(product *core.Product, model tsl.TslData) error {
	{
		// 属性
		var properties map[string]any = map[string]any{}
		for _, p := range model.Properties {
			(properties)[p.Id] = t.createElasticProperty(p)
		}
		properties["deviceId"] = es.Property{Type: "keyword"}
		properties["createTime"] = es.Property{Type: "date", Format: es.DefaultDateFormat}

		indexPattern := fmt.Sprintf("%s-%s-*", properties_const, product.GetId())
		templateName := fmt.Sprintf("%s-%s-template", properties_const, product.GetId())
		err := es.CreateEsTemplate(properties, indexPattern, templateName, "")
		if err != nil {
			return err
		}
	}
	{
		// 事件
		for _, e := range model.Events {
			var properties map[string]any = map[string]any{}
			if object, ok := e.IsObject(); ok {
				for _, p1 := range object.Properties {
					properties[p1.Id] = t.createElasticProperty(p1)
				}
			} else {
				(properties)[e.Id] = t.createElasticProperty(e)
			}
			properties["deviceId"] = es.Property{Type: "keyword"}
			properties["createTime"] = es.Property{Type: "date", Format: es.DefaultDateFormat}

			indexPattern := fmt.Sprintf("%s-%s-%s-*", event_const, product.GetId(), e.Id) // event-{productId}-{eventId}-*
			templateName := fmt.Sprintf("%s-%s-%s-template", event_const, product.GetId(), e.Id)
			err := es.CreateEsTemplate(properties, indexPattern, templateName, "")
			if err != nil {
				return err
			}
		}
	}
	{
		// device logs
		var properties map[string]any = map[string]any{}
		properties["type"] = es.Property{Type: "keyword", IgnoreAbove: "256"}
		properties["traceId"] = es.Property{Type: "keyword"}
		properties["deviceId"] = es.Property{Type: "keyword"}
		properties["content"] = es.Property{Type: "keyword", IgnoreAbove: "256"}
		properties["createTime"] = es.Property{Type: "date", Format: es.DefaultDateFormat}

		indexPattern := fmt.Sprintf("%s-%s-*", devicelogs_const, product.GetId()) // devicelogs-{productId}-{eventId}-*
		templateName := fmt.Sprintf("%s-%s-template", devicelogs_const, product.GetId())
		err := es.CreateEsTemplate(properties, indexPattern, templateName, "")
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *EsTimeSeries) Del(product *core.Product) error {
	now := time.Now()
	return es.DeleteIndex(t.getMonthIndex(product, properties_const, now), t.getMonthIndex(product, event_const, now), t.getMonthIndex(product, devicelogs_const, now))
}

func (t *EsTimeSeries) QueryProperty(product *core.Product, param core.TimeDataSearchRequest) (map[string]any, error) {
	return t.query(t.getIndex(product, properties_const), param)
}

func (t *EsTimeSeries) QueryLogs(product *core.Product, param core.TimeDataSearchRequest) (map[string]any, error) {
	return t.query(t.getIndex(product, devicelogs_const), param)
}

func (t *EsTimeSeries) QueryEvent(product *core.Product, eventId string, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	return t.query(t.getEventIndex(product, event_const, eventId), param)
}

func (t *EsTimeSeries) query(indexName string, param core.TimeDataSearchRequest) (map[string]any, error) {
	if len(param.DeviceId) == 0 {
		return nil, errors.New("deviceId must be persent")
	}
	indexs, err := t.getQueryIndexs(indexName, param)
	if err != nil {
		return nil, err
	}
	filter := es.AppendFilter(param.Condition)
	filter = append(filter, map[string]any{
		"term": map[string]any{tsl.PropertyDeviceId: param.DeviceId},
	})
	if param.PageNum <= 0 {
		param.PageNum = 1
	}
	if param.PageSize <= 0 {
		param.PageSize = 10
	}
	q := es.Query{
		From:        param.PageOffset(),
		Size:        param.PageSize,
		Filter:      filter,
		Sort:        []map[string]es.SortOrder{},
		SearchAfter: param.SearchAfter,
	}
	q.Sort = append(q.Sort, map[string]es.SortOrder{"createTime": {Order: "desc"}})
	es.FilterCount(q, indexs...)
	total, err := es.FilterCount(q, indexs...)
	if err != nil {
		return nil, err
	}
	resp, err := es.FilterSearch(q, indexs...)
	if err != nil {
		return nil, err
	}
	var result map[string]any = map[string]any{
		"pageNum":     param.PageNum,
		"totalCount":  total,
		"list":        []map[string]any{},
		"searchAfter": []any{},
	}
	if err == nil && resp.Total > 0 {
		// convert each hit to result.
		var list []map[string]any = []map[string]any{}
		resp.ConvertSource(&list)
		result["list"] = list
		result["searchAfter"] = resp.LastSort
	}
	return result, nil
}

func (t *EsTimeSeries) SaveProperties(product *core.Product, d1 map[string]any) error {
	validProperty := product.GetTsl().PropertiesMap()
	if validProperty == nil {
		return errors.New("not have tsl property, dont save timeseries data")
	}
	busMsgData := map[string]any{}
	for key := range d1 {
		if key == tsl.PropertyDeviceId {
			continue
		}
		_, ok := validProperty[key]
		if !ok {
			delete(d1, key)
		} else {
			busMsgData[key] = d1[key]
		}
	}
	if len(d1) == 0 {
		return errors.New("data is empty, dont save timeseries data")
	}
	deviceId := d1[tsl.PropertyDeviceId]
	if deviceId == nil {
		return errors.New("not have deviceId, dont save timeseries data")
	}
	d1["createTime"] = time.Now().Format(timeformt)
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Errorf("Error marshaling document: %v", err)
	}

	index := t.getMonthIndex(product, properties_const, time.Now())
	es.Commit(index, string(data))
	// 发送事件总线
	event := eventbus.NewPropertiesMessage(fmt.Sprintf("%v", deviceId), product.GetId(), busMsgData)
	eventbus.PublishProperties(&event)
	return nil
}

func (t *EsTimeSeries) SaveEvents(product *core.Product, eventId string, d1 map[string]any) error {
	eventMap := product.GetTsl().EventsMap()
	if eventMap == nil {
		return errors.New("not have tsl property, dont save timeseries data")
	}
	property, ok := eventMap[eventId]
	if !ok {
		return fmt.Errorf("eventId [%s] not found", eventId)
	}
	busMsgData := map[string]any{}
	columns := []string{}
	if obj, ok := property.IsObject(); ok {
		validProperty := obj.PropertiesMap()
		for key := range d1 {
			if key == tsl.PropertyDeviceId {
				continue
			}
			if _, ok := validProperty[key]; ok {
				columns = append(columns, key)
				busMsgData[key] = d1[key]
			}
		}
	} else {
		for key := range d1 {
			if key == tsl.PropertyDeviceId {
				continue
			}
			columns = append(columns, key)
			busMsgData[key] = d1[key]
		}
	}
	if len(columns) == 0 {
		return errors.New("data is empty, don't save event timeseries data")
	}
	deviceId := d1[tsl.PropertyDeviceId]
	if deviceId == nil {
		return errors.New("not have deviceId, dont save event timeseries data")
	}
	d1["createTime"] = time.Now().Format(timeformt)
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Errorf("Error marshaling document: %v", err)
	}

	index := t.getMonthEventIndex(product, event_const, eventId, time.Now())
	es.Commit(index, string(data))
	// 发送事件总线
	evt := eventbus.NewEventMessage(fmt.Sprintf("%v", deviceId), product.GetId(), eventId, busMsgData)
	eventbus.PublishEvent(&evt)
	return nil
}

func (t *EsTimeSeries) SaveLogs(product *core.Product, d1 core.LogData) error {
	if len(d1.DeviceId) == 0 {
		return errors.New("deviceId must be present, dont save event timeseries data")
	}
	if len(d1.CreateTime) == 0 {
		d1.CreateTime = time.Now().Format(timeformt)
	}
	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Errorf("error marshaling document: %v", err)
	}

	index := t.getMonthIndex(product, devicelogs_const, time.Now())
	es.Commit(index, string(data))
	return nil
}

// goiot-devicelogs-{productId}, goiot-properties-{productId}
func (t *EsTimeSeries) getIndex(product *core.Product, typ string) string {
	index := typ + "-" + product.GetId()
	return index
}

// goiot-devicelogs-{productId}-201102, goiot-properties-{productId}-201102
func (t *EsTimeSeries) getMonthIndex(product *core.Product, typ string, date time.Time) string {
	index := t.getIndex(product, typ) + "-" + date.Format("200601")
	return index
}

// goiot-event-{productId}-{eventId}
func (t *EsTimeSeries) getEventIndex(product *core.Product, typ string, eventId string) string {
	index := typ + "-" + product.GetId() + "-" + eventId
	return index
}

// goiot-event-{productId}-{eventId}-201101
func (t *EsTimeSeries) getMonthEventIndex(product *core.Product, typ string, eventId string, date time.Time) string {
	index := t.getEventIndex(product, typ, eventId) + "-" + date.Format("200601")
	return index
}

// 根据查询时间来列举出索引
func (t *EsTimeSeries) getQueryIndexs(index string, param core.TimeDataSearchRequest) ([]string, error) {
	endTime, _ := time.Parse("2006-01", time.Now().Format("2006-01"))
	startTime := endTime.AddDate(0, -1, 0)
	for _, v := range param.Condition {
		if v.Key == "createTime" {
			s := fmt.Sprintf("%v", v.Value)
			vals := strings.Split(s, ",")
			if len(vals) > 0 {
				sTime, err := time.Parse("2006-01-02 15:04:05", vals[0])
				if err != nil {
					return nil, errors.New("createTime format must be YYYY-MM-dd HH:mm:ss")
				}
				startTime = sTime
				endTime = sTime
			}
			if len(vals) > 1 {
				sTime, err := time.Parse("2006-01-02 15:04:05", vals[1])
				if err != nil {
					return nil, errors.New("createTime format must be YYYY-MM-dd HH:mm:ss")
				}
				endTime = sTime
			}
		}
	}
	indexs := []string{
		index + "-" + startTime.Format("200601"),
	}
	startTime, _ = time.Parse("2006-01", startTime.Format("2006-01"))
	endTime, _ = time.Parse("2006-01", endTime.Format("2006-01"))
	if startTime.Equal(endTime) {
		return indexs, nil
	}
	for startTime.Before(endTime) {
		startTime = startTime.AddDate(0, 1, 0)
		indexs = append(indexs, index+"-"+startTime.Format("200601"))
	}

	return indexs, nil
}

func (t *EsTimeSeries) createElasticProperty(p tsl.TslProperty) any {
	valType := strings.TrimSpace(p.Type)
	switch valType {
	case tsl.TypeInt:
		return es.Property{Type: "integer"}
	case tsl.TypeLong:
		return es.Property{Type: "long"}
	case tsl.TypeFloat:
		return es.Property{Type: "float"}
	case tsl.TypeDouble:
		return es.Property{Type: "double"}
	case tsl.TypeBool:
		return es.Property{Type: "keyword", IgnoreAbove: "256"}
	case tsl.TypeEnum:
		return es.Property{Type: "keyword", IgnoreAbove: "256"}
	case tsl.TypeString:
		return es.Property{Type: "keyword", IgnoreAbove: "256"}
	case tsl.TypePassword:
		return es.Property{Type: "keyword", IgnoreAbove: "256"}
	case tsl.TypeDate:
		return es.Property{Type: "date", Format: es.DefaultDateFormat}
	// case tsl.TypeArray:
	// 	array := p.ValueType.(tsl.ValueTypeArray)
	// 	return t.createElasticProperty(array.ElementType)
	case tsl.TypeObject:
		object := p.ValueType.(tsl.ValueTypeObject)
		var mapping map[string]any = map[string]any{}
		for _, p1 := range object.Properties {
			mapping[p1.Id] = t.createElasticProperty(p1)
		}
		return map[string]any{
			"type":       "object",
			"properties": mapping,
		}
	default:
		if len(p.Id) > 0 {
			return es.Property{Type: "keyword", IgnoreAbove: "256"}
		}
	}
	return nil
}
