package codec

import (
	"bytes"
	"context"
	"encoding/json"
	"go-iot/codec/tsl"

	"github.com/beego/beego/v2/core/logs"
	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/v8"
)

// es时序保存
type EsTimeSeries struct {
}

func (t *EsTimeSeries) Save(product Product, d1 map[string]interface{}) {

	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}

	// Set up the request object.
	req := esapi.IndexRequest{
		Index: "test",
		// DocumentID: "1",
		Body:    bytes.NewReader(data),
		Refresh: "true",
	}
	doRequest(req)
}
func (t *EsTimeSeries) PublishModel(product Product, model tsl.TslData) {
	logs.Info("PublishModel: ", model)
	mapping := t.convertMapping(&model)
	// Build the request body.
	data, err := json.Marshal(mapping)
	if err != nil {
		logs.Error("Error marshaling document: %s", err)
	}
	// Set up the request object.
	req := esapi.IndicesPutMappingRequest{
		Index: []string{product.GetId() + "-*"},
		Body:  bytes.NewReader(data),
	}
	doRequest(req)
}

// 把物模型转换成es mapping
func (t *EsTimeSeries) convertMapping(model *tsl.TslData) map[string]interface{} {
	props := model.Properties
	var properties map[string]interface{} = map[string]interface{}{}
	for _, p := range props {
		valType := p.ValueType["type"]
		esType := ""
		switch valType {
		case "enum":
			esType = "keyword"
		case "int":
			esType = "long"
		case "string":
			esType = "keyword"
		case "float":
			esType = "float"
		case "double":
			esType = "double"
		case "bool":
			esType = "boolean"
		default:
			esType = "keyword"
		}
		properties[p.Name] = struct{ Type string }{Type: esType}
	}
	var mapping map[string]interface{} = map[string]interface{}{}
	mapping["properties"] = properties
	return mapping
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

func doRequest(s esDo) {
	es, err := getEsClient()
	if err != nil {
		logs.Error("Error creating the client: %s", err)
	}
	// Perform the request with the client.
	res, err := s.Do(context.Background(), es)
	if err != nil {
		logs.Error("Error getting response: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		logs.Error("[%s] Error:[%s]", res.Status(), res.String())
	} else {
		// Deserialize the response into a map.
		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			logs.Error("Error parsing the response body: %s", err)
		} else {
			// Print the response status and indexed document version.
			logs.Info("[%s] %s; version=%d", res.Status(), r["result"], int(r["_version"].(float64)))
		}
	}
}
