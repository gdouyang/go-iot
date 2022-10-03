package codec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go-iot/codec/tsl"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// es时序保存
type EsTimeSeries struct {
}

func (t *EsTimeSeries) Save(product Product, d1 map[string]interface{}) error {

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

func (t *EsTimeSeries) QueryProperty(product Product) (map[string]interface{}, error) {
	req := esapi.SearchRequest{
		Index: []string{t.getIndex(product)},
	}
	r, err := doRequest(req)
	var resp map[string]interface{} = map[string]interface{}{}
	if err != nil {
		total := int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
		resp["total"] = total
		// Print the ID and document source for each hit.
		var list []map[string]interface{}
		for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
			d := (hit.(map[string]interface{})["_source"].(map[string]interface{}))
			list = append(list, d)
		}
		resp["list"] = list
	}
	return resp, err
}

func (t *EsTimeSeries) getIndex(product Product) string {
	index := product.GetId() + "-" + time.Now().Format("20060102")
	return index
}

// 把物模型转换成es mapping
func (t *EsTimeSeries) convertMapping(product Product, model *tsl.TslData) map[string]interface{} {
	props := model.Properties
	var mapping map[string]interface{} = map[string]interface{}{}
	for _, p := range props {
		valType := p.ValueType["type"]
		esType := ""
		switch valType {
		case tsl.VALUE_TYPE_ENUM:
			esType = "keyword"
		case tsl.VALUE_TYPE_INT:
			esType = "long"
		case tsl.VALUE_TYPE_STRING:
			esType = "keyword"
		case tsl.VALUE_TYPE_FLOAT:
			esType = "float"
		case tsl.VALUE_TYPE_DOUBLE:
			esType = "double"
		case tsl.VALUE_TYPE_BOOL:
			esType = "boolean"
		default:
			esType = "keyword"
		}
		mapping[p.Name] = struct {
			Type string `json:"type"`
		}{Type: esType}
	}
	var payload map[string]interface{} = map[string]interface{}{
		"index_patterns": []string{product.GetId() + "-*"},
		"order":          0,
		// "template": map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards": 1,
		},
		"mappings": map[string]interface{}{
			"properties": mapping,
		},
		// },
	}
	return payload
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
