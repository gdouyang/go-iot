package codec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/codec/eventbus"
	"go-iot/codec/tsl"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func init() {
	timeSeriseMap[TIME_SERISE_ES] = &EsTimeSeries{dataCh: make(chan string, 1000)}
}

const (
	properties_const = "properties"
	event_const      = "event"
	devicelogs_const = "devicelogs"
)

var ES_URL string = "http://localhost:9200"
var ES_PASSWORD string = ""
var ES_USERNAME string = ""

// es时序保存
type EsTimeSeries struct {
	sync.RWMutex
	batchData    []string
	dataCh       chan string
	batchTaskRun bool
}

func (t *EsTimeSeries) PublishModel(product Product, model tsl.TslData) error {
	logs.Info("PublishModel: ", model)
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

func (t *EsTimeSeries) QueryProperty(product Product, param QueryParam) (map[string]interface{}, error) {
	if len(param.DeviceId) == 0 {
		return nil, errors.New("deviceId must be persent")
	}
	if param.Type != properties_const && param.Type != event_const && param.Type != devicelogs_const {
		return nil, fmt.Errorf("type is invalid, must be [%s, %s, %s]", properties_const, event_const, devicelogs_const)
	}
	body := map[string]interface{}{
		"from": param.PageNum,
		"size": param.PageSize,
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				tsl.PropertyDeviceId: param.DeviceId,
			},
		},
		"sort": []map[string]interface{}{
			{
				"collectTime_": map[string]interface{}{
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
		var list []map[string]interface{}
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

func (t *EsTimeSeries) SaveProperties(product Product, d1 map[string]interface{}) error {
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
	d1["collectTime_"] = time.Now().Format("2006-01-02 15:04:05.000")
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

func (t *EsTimeSeries) SaveEvents(product Product, eventId string, d1 map[string]interface{}) error {
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
	d1["collectTime_"] = time.Now().Format("2006-01-02 15:04:05.000")
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

func (t *EsTimeSeries) SaveLogs(product Product, d1 LogData) error {
	if len(d1.DeviceId) == 0 {
		return errors.New("deviceId must be present, dont save event timeseries data")
	}
	d1.CollectTime_ = time.Now().Format("2006-01-02 15:04:05.000")
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
	if !t.batchTaskRun {
		t.Lock()
		defer t.Unlock()
		t.batchTaskRun = true
		go t.batchSave()
	}
}

func (t *EsTimeSeries) batchSave() {
	for {
		select {
		case <-time.After(time.Duration(1) * time.Millisecond * 10000):
			t.save()
		case d := <-t.dataCh:
			t.batchData = append(t.batchData, d)
			if len(t.batchData) >= 10 {
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
		doRequest(req)
	}
}

func (t *EsTimeSeries) getIndex(product Product, typ string) string {
	index := typ + "-" + product.GetId() + "-" + time.Now().Format("200601")
	return index
}

// 把物模型转换成es mapping
func (t *EsTimeSeries) propertiesTplMapping(product Product, model *tsl.TslData) error {
	var properties map[string]interface{} = map[string]interface{}{}
	for _, p := range model.Properties {
		(properties)[p.Id] = t.createElasticProperty(p)
	}
	properties["deviceId"] = esType{Type: "keyword"}
	properties["collectTime_"] = esType{Type: "date", Format: defaultDateFormat}

	var payload map[string]interface{} = map[string]interface{}{
		"index_patterns": []string{fmt.Sprintf("%s-%s-*", properties_const, product.GetId())},
		"order":          0,
		// "template": map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   "1",
			"number_of_replicas": "0",
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

func (t *EsTimeSeries) eventsTplMapping(product Product, model *tsl.TslData) error {
	var properties map[string]interface{} = map[string]interface{}{}
	for _, e := range model.Events {
		for _, p := range e.Properties {
			(properties)[p.Id] = t.createElasticProperty(p)
		}
		properties["deviceId"] = esType{Type: "keyword"}
		properties["collectTime_"] = esType{Type: "date", Format: defaultDateFormat}

		var payload map[string]interface{} = map[string]interface{}{
			"index_patterns": []string{fmt.Sprintf("%s-%s-%s-*", event_const, product.GetId(), e.Id)}, // event-{productId}-{eventId}-*
			"order":          0,
			// "template": map[string]interface{}{
			"settings": map[string]interface{}{
				"number_of_shards":   "1",
				"number_of_replicas": "0",
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

func (t *EsTimeSeries) logsTplMapping(product Product) error {
	var properties map[string]interface{} = map[string]interface{}{}
	properties["deviceId"] = esType{Type: "keyword"}
	properties["type"] = esType{Type: "keyword", IgnoreAbove: "512"}
	properties["content"] = esType{Type: "keyword", IgnoreAbove: "512"}
	properties["collectTime_"] = esType{Type: "date", Format: defaultDateFormat}

	var payload map[string]interface{} = map[string]interface{}{
		"index_patterns": []string{fmt.Sprintf("%s-%s-*", devicelogs_const, product.GetId())}, // devicelogs-{productId}-{eventId}-*
		"order":          0,
		// "template": map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   "1",
			"number_of_replicas": "0",
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
		return esType{Type: "int"}
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

type esDo interface {
	Do(ctx context.Context, transport esapi.Transport) (*esapi.Response, error)
}

func getEsClient() (*elasticsearch.Client, error) {
	// Initialize a client with the default settings.
	//
	// An `ELASTICSEARCH_URL` environment variable will be used when exported.
	//
	addrs := strings.Split(ES_URL, ",")
	config := elasticsearch.Config{
		Addresses: addrs,
	}
	if len(ES_USERNAME) > 0 {
		config.Username = ES_USERNAME
		config.Password = ES_PASSWORD
	}
	es, err := elasticsearch.NewClient(config)
	return es, err
}

func doRequest(s esDo) (map[string]interface{}, error) {
	es, err := getEsClient()
	if err != nil {
		logs.Error("Error creating the client: %s", err)
	}
	// Perform the request with the client.
	res, err := s.Do(context.Background(), es)
	if err != nil {
		logs.Error("Error getting response: %s", err)
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, nil
	}

	if res.IsError() {
		logs.Error("[%s] Error:[%s]", res.Status(), res.String())
		return nil, errors.New(res.String())
	} else {
		// Deserialize the response into a map.
		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			logs.Error("Error parsing the response body: %s", err)
		} else {
			return r, nil
		}
	}
	return nil, err
}
