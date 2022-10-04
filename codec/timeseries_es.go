package codec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go-iot/codec/tsl"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// es时序保存
type EsTimeSeries struct {
}

func (t *EsTimeSeries) Save(product Product, d1 map[string]interface{}) error {
	validProperty := product.GetTslProperty()
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
	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}

	index := t.getIndex(product)
	// Set up the request object.
	req := esapi.IndexRequest{
		Index: index,
		// DocumentID: "1",
		Body:    bytes.NewReader(data),
		Refresh: "true",
	}
	_, err = doRequest(req)
	return err
}
func (t *EsTimeSeries) PublishModel(product Product, model tsl.TslData) error {
	logs.Info("PublishModel: ", model)
	mapping := t.convertMapping(product, &model)
	// Build the request body.
	data, err := json.Marshal(mapping)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}
	logs.Info(string(data))
	// Set up the request object.
	req := esapi.IndicesPutTemplateRequest{
		Name: product.GetId() + "-month-tpl",
		Body: bytes.NewReader(data),
	}
	_, err = doRequest(req)
	return err
}

func (t *EsTimeSeries) QueryProperty(product Product, param map[string]interface{}) (map[string]interface{}, error) {
	if _, ok := param[tsl.PropertyDeviceId]; !ok {
		return nil, errors.New("deviceId property not persent")
	}
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				tsl.PropertyDeviceId: param[tsl.PropertyDeviceId],
			},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req := esapi.SearchRequest{
		Index: []string{t.getIndex(product)},
		Body:  bytes.NewReader(data),
	}
	r, err := doRequest(req)
	var resp map[string]interface{} = map[string]interface{}{}
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

func (t *EsTimeSeries) getIndex(product Product) string {
	index := product.GetId() + "-" + time.Now().Format("200601")
	return index
}

// 把物模型转换成es mapping
func (t *EsTimeSeries) convertMapping(product Product, model *tsl.TslData) map[string]interface{} {
	var properties map[string]interface{} = map[string]interface{}{}
	for _, p := range model.Properties {
		valType := strings.TrimSpace(p.ValueType["type"].(string))
		type1 := ""
		switch valType {
		case tsl.TypeEnum:
			type1 = "keyword"
		case tsl.TypeInt:
			type1 = "long"
		case tsl.TypeString:
			type1 = "keyword"
		case tsl.TypeFloat:
			type1 = "float"
		case tsl.TypeDouble:
			type1 = "double"
		case tsl.TypeBool:
			type1 = "boolean"
		case tsl.TypeDate:
			type1 = "date"
		default:
			type1 = "keyword"
		}
		properties[p.Id] = esType{Type: type1}
	}
	properties["deviceId"] = esType{Type: "keyword"}

	var payload map[string]interface{} = map[string]interface{}{
		"index_patterns": []string{product.GetId() + "-*"},
		"order":          0,
		// "template": map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards": 1,
		},
		"mappings": map[string]interface{}{
			"properties": properties,
		},
		// },
	}
	return payload
}

type esType struct {
	Type string `json:"type"`
}

type esDo interface {
	Do(ctx context.Context, transport esapi.Transport) (*esapi.Response, error)
}

func getEsClient() (*elasticsearch.Client, error) {
	// Initialize a client with the default settings.
	//
	// An `ELASTICSEARCH_URL` environment variable will be used when exported.
	//
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
		// Username:  "username",
		// Password:  "password",
	})
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
