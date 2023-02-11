package es

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

var DefaultEsSaveHelper EsDataSaveHelper = EsDataSaveHelper{dataCh: make(chan string, DefaultEsConfig.BufferSize)}

type EsDataSaveHelper struct {
	sync.RWMutex
	batchData    []string
	dataCh       chan string
	batchTaskRun bool
}

func Commit(index string, text string) {
	DefaultEsSaveHelper.commit(index, text)
}

func (t *EsDataSaveHelper) commit(index string, text string) {
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

func (t *EsDataSaveHelper) batchSave() {
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

func (t *EsDataSaveHelper) save() {
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
		DoRequest[map[string]interface{}](req)
		totalTime := time.Now().UnixMilli() - start
		if DefaultEsConfig.WarnTime > 0 && totalTime > int64(DefaultEsConfig.WarnTime) {
			logs.Warn("save data to es use time: %v ms", totalTime)
		}
	}
}

type EsType struct {
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
	addrs := strings.Split(DefaultEsConfig.Url, ",")
	config := elasticsearch.Config{
		Addresses: addrs,
	}
	if len(DefaultEsConfig.Username) > 0 {
		config.Username = DefaultEsConfig.Username
		config.Password = DefaultEsConfig.Password
	}
	es, err := elasticsearch.NewClient(config)
	return es, err
}

func DoRequest[T any](s esDo) (T, error) {
	var result T
	es, err := getEsClient()
	if err != nil {
		logs.Error("Error creating the client: %s", err)
	}
	// Perform the request with the client.
	res, err := s.Do(context.Background(), es)
	if err != nil {
		logs.Error("Error getting response: %s", err)
		return result, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return result, nil
	}

	if res.IsError() {
		logs.Error("[%s] Error:[%s]", res.Status(), res.String())
		return result, errors.New(res.String())
	} else {
		// Deserialize the response into a map.
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			logs.Error("Error parsing the response body: %s", err)
		} else {
			return result, nil
		}
	}
	return result, err
}

func AppendFilter(condition map[string]interface{}) []map[string]interface{} {
	filter := []map[string]interface{}{}
	for key, val := range condition {
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
	return filter
}
