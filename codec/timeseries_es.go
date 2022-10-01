package codec

import (
	"bytes"
	"context"
	"encoding/json"
	"go-iot/codec/tsl"
	"log"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/v8"
)

// es时序保存
type EsTimeSeries struct {
}

func (t *EsTimeSeries) Save(productId string, d1 map[string]interface{}) {
	// Initialize a client with the default settings.
	//
	// An `ELASTICSEARCH_URL` environment variable will be used when exported.
	//
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
		// Username:  "username",
		// Password:  "password",
	})
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}
	// 2. Index documents concurrently
	//
	// Build the request body.
	data, err := json.Marshal(d1)
	if err != nil {
		log.Fatalf("Error marshaling document: %s", err)
	}

	// Set up the request object.
	id := time.Now().UnixMilli()
	req := esapi.IndexRequest{
		Index:      "test",
		DocumentID: "",
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}

	// Perform the request with the client.
	res, err := req.Do(context.Background(), es)
	if err != nil {
		logs.Error("Error getting response: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		logs.Error("[%s] Error indexing document ID=%d", res.Status(), id)
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
func (t *EsTimeSeries) PublishModel(product string, model tsl.TslData) {
	logs.Info("PublishModel: ", model)
}
