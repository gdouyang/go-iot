package codec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/codec/eventbus"
	"go-iot/pkg/codec/tsl"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func init() {
	RegEsTimeSeries()
}

func RegEsTimeSeries() {
	timeSeriseMap[TIME_SERISE_ES] = &EsTimeSeries{dataCh: make(chan string, DefaultEsConfig.BufferSize)}
}

const (
	properties_const = "properties"
	event_const      = "event"
	devicelogs_const = "devicelogs"
)

// es时序保存
type EsTimeSeries struct {
	sync.RWMutex
	batchData    []string
	dataCh       chan string
	batchTaskRun bool
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
	_, err := doRequest(req)
	return err
}

func (t *EsTimeSeries) QueryProperty(product *Product, param QueryParam) (map[string]interface{}, error) {
	if len(param.DeviceId) == 0 {
		return nil, errors.New("deviceId must be persent")
	}
	if param.Type != properties_const && param.Type != event_const && param.Type != devicelogs_const {
		return nil, fmt.Errorf("type is invalid, must be [%s, %s, %s]", properties_const, event_const, devicelogs_const)
	}
	filter := []map[string]interface{}{
		{"term": map[string]interface{}{
			tsl.PropertyDeviceId: param.DeviceId,
		},
		}}
	for key, val := range param.Condition {
		s := fmt.Sprintf("%v", val)
		if len(strings.TrimSpace(s)) > 0 && s != "<nil>" {
			var term map[string]interface{} = map[string]interface{}{}
			if strings.Contains(key, "$IN") {
				prop := strings.ReplaceAll(key, "$IN", "")
				term["terms"] = map[string]interface{}{prop: strings.Split(s, ",")}
			} else if strings.Contains(key, "$BTW") {
				prop := strings.ReplaceAll(key, "$BTW", "")
				vals := strings.Split(s, ",")
				if len(vals) < 1 {
					continue
				}
				term["range"] = map[string]interface{}{prop: map[string]interface{}{
					"gte": vals[0],
					"lte": vals[1],
				}}
			} else {
				term["term"] = map[string]interface{}{key: val}
			}
			filter = append(filter, term)
		}
	}
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
	r, err := doRequest(req)
	var resp map[string]interface{} = map[string]interface{}{
		"pageNum": param.PageNum,
	}
	if err == nil && r != nil {
		total := int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
		resp["totalCount"] = total
		// convert each hit to result.
		var list []map[string]interface{} = []map[string]interface{}{}
		for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
			d := (hit.(map[string]interface{})["_source"].(map[string]interface{}))
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
	d1["createTime"] = time.Now().Format("2006-01-02 15:04:05.000")
	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}

	index := t.getIndex(product, properties_const)
	t.commit(index, string(data))
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
	d1["createTime"] = time.Now().Format("2006-01-02 15:04:05.000")
	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}

	index := t.getIndex(product, event_const)
	t.commit(index, string(data))
	evt := eventbus.NewEventMessage(fmt.Sprintf("%v", deviceId), product.GetId(), d1)
	eventbus.PublishEvent(&evt)
	return nil
}

func (t *EsTimeSeries) SaveLogs(product *Product, d1 LogData) error {
	if len(d1.DeviceId) == 0 {
		return errors.New("deviceId must be present, dont save event timeseries data")
	}
	d1.CreateTime = time.Now().Format("2006-01-02 15:04:05.000")
	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}

	index := t.getIndex(product, devicelogs_const)
	t.commit(index, string(data))
	return nil
}

func (t *EsTimeSeries) commit(index string, text string) {
	o := `{ "index" : { "_index" : "` + index + `" } }` + "\n" + text + "\n"
	t.dataCh <- o
	if len(t.dataCh) > (DefaultEsConfig.BufferSize / 2) {
		logs.Info("commit data to es, chan length:", len(t.dataCh))
	}
	if !t.batchTaskRun {
		t.Lock()
		defer t.Unlock()
		if !t.batchTaskRun {
			t.batchTaskRun = true
			go t.batchSave()
		}
	}
}

func (t *EsTimeSeries) batchSave() {
	for {
		select {
		case <-time.After(time.Millisecond * 5000): // every 5 sec save data
			t.save()
		case d := <-t.dataCh:
			t.batchData = append(t.batchData, d)
			if len(t.batchData) >= DefaultEsConfig.BulkSize {
				t.save()
			}
		}
	}
}

func (t *EsTimeSeries) save() {
	if len(t.batchData) > 0 {
		var data []byte
		for i := 0; i < len(t.batchData); i++ {
			data = append(data, t.batchData[i]...)
		}
		// clear batch data
		t.batchData = t.batchData[:0]
		req := esapi.BulkRequest{
			Body: bytes.NewReader(data),
		}
		start := time.Now().UnixMilli()
		doRequest(req)
		totalTime := time.Now().UnixMilli() - start
		if DefaultEsConfig.WarnTime > 0 && totalTime > int64(DefaultEsConfig.WarnTime) {
			logs.Warn("save data to es use time: %v ms", totalTime)
		}
	}
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
	properties["deviceId"] = esType{Type: "keyword"}
	properties["createTime"] = esType{Type: "date", Format: defaultDateFormat}

	var payload map[string]interface{} = map[string]interface{}{
		"index_patterns": []string{fmt.Sprintf("%s-%s-*", properties_const, product.GetId())},
		"order":          0,
		// "template": map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   DefaultEsConfig.NumberOfShards,
			"number_of_replicas": DefaultEsConfig.NumberOfReplicas,
		},
		"mappings": map[string]interface{}{
			"dynamic":    false,
			"properties": properties,
		},
		// },
	}
	// Build the request body.
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("propertiesTplMapping error: %s", err.Error())
	}
	logs.Info(string(data))
	// Set up the request object.
	req := esapi.IndicesPutTemplateRequest{
		Name: fmt.Sprintf("properties_%s_template", product.GetId()),
		Body: bytes.NewReader(data),
	}
	_, err = doRequest(req)
	return err
}

func (t *EsTimeSeries) eventsTplMapping(product *Product, model *tsl.TslData) error {
	var properties map[string]interface{} = map[string]interface{}{}
	for _, e := range model.Events {
		for _, p := range e.Properties {
			(properties)[p.Id] = t.createElasticProperty(p)
		}
		properties["deviceId"] = esType{Type: "keyword"}
		properties["createTime"] = esType{Type: "date", Format: defaultDateFormat}

		var payload map[string]interface{} = map[string]interface{}{
			"index_patterns": []string{fmt.Sprintf("%s-%s-%s-*", event_const, product.GetId(), e.Id)}, // event-{productId}-{eventId}-*
			"order":          0,
			// "template": map[string]interface{}{
			"settings": map[string]interface{}{
				"number_of_shards":   DefaultEsConfig.NumberOfShards,
				"number_of_replicas": DefaultEsConfig.NumberOfReplicas,
			},
			"mappings": map[string]interface{}{
				"dynamic":    false,
				"properties": properties,
			},
			// },
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("eventsTplMapping error: %s", err.Error())
		}
		logs.Info(string(data))
		// Set up the request object.
		req := esapi.IndicesPutTemplateRequest{
			Name: fmt.Sprintf("event_%s_%s_template", product.GetId(), e.Id),
			Body: bytes.NewReader(data),
		}
		_, err = doRequest(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *EsTimeSeries) logsTplMapping(product *Product) error {
	var properties map[string]interface{} = map[string]interface{}{}
	properties["deviceId"] = esType{Type: "keyword"}
	properties["type"] = esType{Type: "keyword", IgnoreAbove: "512"}
	properties["content"] = esType{Type: "keyword", IgnoreAbove: "512"}
	properties["createTime"] = esType{Type: "date", Format: defaultDateFormat}

	var payload map[string]interface{} = map[string]interface{}{
		"index_patterns": []string{fmt.Sprintf("%s-%s-*", devicelogs_const, product.GetId())}, // devicelogs-{productId}-{eventId}-*
		"order":          0,
		// "template": map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   DefaultEsConfig.NumberOfShards,
			"number_of_replicas": DefaultEsConfig.NumberOfReplicas,
		},
		"mappings": map[string]interface{}{
			"dynamic":    false,
			"properties": properties,
		},
		// },
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("eventsTplMapping error: %s", err.Error())
	}
	logs.Info(string(data))
	// Set up the request object.
	req := esapi.IndicesPutTemplateRequest{
		Name: fmt.Sprintf("devicelogs_%s_template", product.GetId()),
		Body: bytes.NewReader(data),
	}
	_, err = doRequest(req)
	if err != nil {
		return err
	}
	return nil
}

func (t *EsTimeSeries) createElasticProperty(p tsl.TslProperty) interface{} {
	valType := strings.TrimSpace(p.Type)
	switch valType {
	case tsl.TypeInt:
		return esType{Type: "integer"}
	case tsl.TypeLong:
		return esType{Type: "long"}
	case tsl.TypeFloat:
		return esType{Type: "float"}
	case tsl.TypeDouble:
		return esType{Type: "double"}
	case tsl.TypeBool:
		return esType{Type: "keyword", IgnoreAbove: "512"}
	case tsl.TypeEnum:
		return esType{Type: "keyword", IgnoreAbove: "512"}
	case tsl.TypeString:
		return esType{Type: "keyword", IgnoreAbove: "512"}
	case tsl.TypePassword:
		return esType{Type: "keyword", IgnoreAbove: "512"}
	case tsl.TypeDate:
		return esType{Type: "date", Format: defaultDateFormat}
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
			return esType{Type: "keyword", IgnoreAbove: "512"}
		}
	}
	return nil
}

const defaultDateFormat string = "yyyy-MM||yyyy-MM-dd||yyyy-MM-dd HH:mm:ss||yyyy-MM-dd HH:mm:ss.SSS||epoch_millis"

type esType struct {
	Type        string `json:"type"`
	IgnoreAbove string `json:"ignore_above,omitempty"`
	Format      string `json:"format,omitempty"`
}