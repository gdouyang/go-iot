package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/core/es"
	"go-iot/pkg/core/eventbus"
	"go-iot/pkg/core/tsl"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func init() {
	RegEsTimeSeries(false)
}

func RegEsTimeSeries(docheck bool) {
	timeSeriseMap[TIME_SERISE_ES] = &EsTimeSeries{}
}

const (
	prefix           = "goiot-"
	properties_const = prefix + "properties"
	event_const      = prefix + "event"
	devicelogs_const = prefix + "devicelogs"
	timeformt        = "2006-01-02 15:04:05.000"
)

// es时序保存
type EsTimeSeries struct {
}

func (t *EsTimeSeries) PublishModel(product *Product, model tsl.TslData) error {
	err := t.propertiesTplMapping(product, &model)
	if err != nil {
		return err
	}
	err = t.eventsTplMapping(product, &model)
	if err != nil {
		return err
	}
	err = t.logsTplMapping(product)
	if err != nil {
		return err
	}
	return err
}

func (t *EsTimeSeries) Del(product *Product) error {
	var IgnoreUnavailable bool = true
	req := esapi.IndicesDeleteRequest{
		Index:             []string{t.getIndex(product, properties_const), t.getIndex(product, event_const), t.getIndex(product, devicelogs_const)},
		IgnoreUnavailable: &IgnoreUnavailable,
	}
	_, err := es.DoRequest[map[string]interface{}](req)
	if err != nil {
		return err.OriginErr
	}
	return nil
}

func (t *EsTimeSeries) QueryProperty(product *Product, param QueryParam) (map[string]interface{}, error) {
	if len(param.DeviceId) == 0 {
		return nil, errors.New("deviceId must be persent")
	}
	if !strings.HasPrefix(param.Type, prefix) {
		param.Type = prefix + param.Type
	}
	if param.Type != properties_const && param.Type != event_const && param.Type != devicelogs_const {
		return nil, errors.New("type is invalid, must be [properties, event, devicelogs]")
	}
	filter := es.AppendFilter(param.Condition)
	filter = append(filter, map[string]interface{}{
		"term": map[string]interface{}{tsl.PropertyDeviceId: param.DeviceId},
	})
	if param.PageNum <= 0 {
		param.PageNum = 1
	}
	if param.PageSize <= 0 {
		param.PageSize = 10
	}
	body := map[string]interface{}{
		"from": param.PageOffset(),
		"size": param.PageSize,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filter,
			},
		},
		"sort": []map[string]interface{}{
			{
				"createTime": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req := esapi.SearchRequest{
		Index: []string{t.getIndex(product, param.Type)},
		Body:  bytes.NewReader(data),
	}
	r, eserr := es.DoRequest[es.EsQueryResult[map[string]interface{}]](req)
	if eserr != nil {
		return nil, eserr.OriginErr
	}
	var resp map[string]interface{} = map[string]interface{}{
		"pageNum": param.PageNum,
	}
	if err == nil && len(r.Hits.Hits) > 0 {
		total := r.Hits.Total.Value
		resp["totalCount"] = total
		// convert each hit to result.
		var list []map[string]interface{} = []map[string]interface{}{}
		for _, hit := range r.Hits.Hits {
			d := hit.Source
			list = append(list, d)
		}
		resp["list"] = list
	} else {
		resp["totalCount"] = 0
		resp["list"] = []map[string]interface{}{}
	}
	return resp, err
}

func (t *EsTimeSeries) SaveProperties(product *Product, d1 map[string]interface{}) error {
	validProperty := product.GetTsl().PropertiesMap()
	if validProperty == nil {
		return errors.New("not have tsl property, dont save timeseries data")
	}
	for key := range d1 {
		if key == tsl.PropertyDeviceId {
			continue
		}
		if _, ok := validProperty[key]; !ok {
			delete(d1, key)
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
	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}

	index := t.getIndex(product, properties_const)
	es.Commit(index, string(data))
	event := eventbus.NewPropertiesMessage(fmt.Sprintf("%v", deviceId), product.GetId(), d1)
	eventbus.PublishProperties(&event)
	return nil
}

func (t *EsTimeSeries) SaveEvents(product *Product, eventId string, d1 map[string]interface{}) error {
	validProperty := product.GetTsl().EventsMap()
	if validProperty == nil {
		return errors.New("not have tsl property, dont save timeseries data")
	}
	event, ok := validProperty[eventId]
	if !ok {
		return fmt.Errorf("eventId [%s] not found", eventId)
	}
	emap := event.PropertiesMap()
	for key := range d1 {
		if key == tsl.PropertyDeviceId {
			continue
		}
		if _, ok := emap[key]; !ok {
			delete(d1, key)
		}
	}
	if len(d1) == 0 {
		return errors.New("data is empty, dont save timeseries data")
	}
	deviceId := d1[tsl.PropertyDeviceId]
	if deviceId == nil {
		return errors.New("not have deviceId, dont save event timeseries data")
	}
	d1["createTime"] = time.Now().Format(timeformt)
	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}

	index := t.getIndex(product, event_const)
	es.Commit(index, string(data))
	evt := eventbus.NewEventMessage(fmt.Sprintf("%v", deviceId), product.GetId(), d1)
	eventbus.PublishEvent(&evt)
	return nil
}

func (t *EsTimeSeries) SaveLogs(product *Product, d1 LogData) error {
	if len(d1.DeviceId) == 0 {
		return errors.New("deviceId must be present, dont save event timeseries data")
	}
	if len(d1.CreateTime) == 0 {
		d1.CreateTime = time.Now().Format(timeformt)
	}
	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}

	index := t.getIndex(product, devicelogs_const)
	es.Commit(index, string(data))
	return nil
}

func (t *EsTimeSeries) getIndex(product *Product, typ string) string {
	index := typ + "-" + product.GetId() + "-" + time.Now().Format("200601")
	return index
}

// 把物模型转换成es mapping
func (t *EsTimeSeries) propertiesTplMapping(product *Product, model *tsl.TslData) error {
	var properties map[string]interface{} = map[string]interface{}{}
	for _, p := range model.Properties {
		(properties)[p.Id] = t.createElasticProperty(p)
	}
	properties["deviceId"] = es.EsType{Type: "keyword"}
	properties["createTime"] = es.EsType{Type: "date", Format: es.DefaultDateFormat}

	indexPattern := fmt.Sprintf("%s-%s-*", properties_const, product.GetId())
	templateName := fmt.Sprintf("%s_%s_template", properties_const, product.GetId())
	err := es.CreateEsTemplate(properties, indexPattern, templateName)
	return err
}

func (t *EsTimeSeries) eventsTplMapping(product *Product, model *tsl.TslData) error {
	for _, e := range model.Events {
		var properties map[string]interface{} = map[string]interface{}{}
		for _, p := range e.Properties {
			(properties)[p.Id] = t.createElasticProperty(p)
		}
		properties["deviceId"] = es.EsType{Type: "keyword"}
		properties["createTime"] = es.EsType{Type: "date", Format: es.DefaultDateFormat}

		indexPattern := fmt.Sprintf("%s-%s-%s-*", event_const, product.GetId(), e.Id) // event-{productId}-{eventId}-*
		templateName := fmt.Sprintf("%s_%s_%s_template", event_const, product.GetId(), e.Id)
		err := es.CreateEsTemplate(properties, indexPattern, templateName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *EsTimeSeries) logsTplMapping(product *Product) error {
	var properties map[string]interface{} = map[string]interface{}{}
	properties["deviceId"] = es.EsType{Type: "keyword"}
	properties["type"] = es.EsType{Type: "keyword", IgnoreAbove: "256"}
	properties["content"] = es.EsType{Type: "keyword", IgnoreAbove: "256"}
	properties["createTime"] = es.EsType{Type: "date", Format: es.DefaultDateFormat}

	indexPattern := fmt.Sprintf("%s-%s-*", devicelogs_const, product.GetId()) // devicelogs-{productId}-{eventId}-*
	templateName := fmt.Sprintf("%s_%s_template", devicelogs_const, product.GetId())
	err := es.CreateEsTemplate(properties, indexPattern, templateName)
	return err
}

func (t *EsTimeSeries) createElasticProperty(p tsl.TslProperty) interface{} {
	valType := strings.TrimSpace(p.Type)
	switch valType {
	case tsl.TypeInt:
		return es.EsType{Type: "integer"}
	case tsl.TypeLong:
		return es.EsType{Type: "long"}
	case tsl.TypeFloat:
		return es.EsType{Type: "float"}
	case tsl.TypeDouble:
		return es.EsType{Type: "double"}
	case tsl.TypeBool:
		return es.EsType{Type: "keyword", IgnoreAbove: "256"}
	case tsl.TypeEnum:
		return es.EsType{Type: "keyword", IgnoreAbove: "256"}
	case tsl.TypeString:
		return es.EsType{Type: "keyword", IgnoreAbove: "256"}
	case tsl.TypePassword:
		return es.EsType{Type: "keyword", IgnoreAbove: "256"}
	case tsl.TypeDate:
		return es.EsType{Type: "date", Format: es.DefaultDateFormat}
	case tsl.TypeArray:
		array := p.ValueType.(tsl.ValueTypeArray)
		return t.createElasticProperty(array.ElementType)
	case tsl.TypeObject:
		object := p.ValueType.(tsl.ValueTypeObject)
		var mapping map[string]interface{} = map[string]interface{}{}
		for _, p1 := range object.Properties {
			mapping[p1.Id] = t.createElasticProperty(p1)
		}
		return map[string]interface{}{
			"type":       "nested",
			"properties": mapping,
		}
	default:
		if len(p.Id) > 0 {
			return es.EsType{Type: "keyword", IgnoreAbove: "256"}
		}
	}
	return nil
}
