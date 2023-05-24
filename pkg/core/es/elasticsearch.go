package es

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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

func DoRequest[T any](s esDo) (T, *EsErrorResult) {
	var result T
	es, err := getEsClient()
	if err != nil {
		logs.Error("Error creating the client: %s", err)
	}
	// Perform the request with the client.
	res, err := s.Do(context.Background(), es)
	if err != nil {
		logs.Error("Error getting response: %s", err)
		return result, NewEsError(err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return result, nil
	}

	if res.IsError() {
		// Deserialize the response into a map.
		var eserr *EsErrorResult = &EsErrorResult{OriginErr: errors.New(res.String())}
		if err := json.NewDecoder(res.Body).Decode(eserr); err != nil {
			logs.Error("Error parsing the response body: %s", err)
			logs.Error("[%s] Error:[%s]", res.Status(), res.String())
		} else {
			return result, eserr
		}
	} else {
		// Deserialize the response into a map.
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		s := buf.String()
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			logs.Error("Error parsing the response body: %s, err: %s", s, err)
		} else {
			return result, nil
		}
	}
	if err != nil {
		return result, NewEsError(err)
	}
	return result, nil
}

func CreateEsTemplate(properties map[string]interface{}, indexPattern string, templateName string, refresh_interval string) error {
	if len(refresh_interval) == 0 {
		refresh_interval = "10s"
	}
	settings := map[string]interface{}{
		"number_of_shards":   DefaultEsConfig.NumberOfShards,
		"number_of_replicas": DefaultEsConfig.NumberOfReplicas,
	}
	if len(refresh_interval) > 0 {
		settings["refresh_interval"] = refresh_interval
	}
	var payload map[string]interface{} = map[string]interface{}{
		"index_patterns": []string{indexPattern},
		"order":          0,
		// "template": map[string]interface{}{
		"settings": settings,
		"mappings": map[string]interface{}{
			"dynamic":    false,
			"properties": properties,
		},
		// },
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s error: %s", templateName, err.Error())
	}
	logs.Info(string(data))
	// Set up the request object.
	req := esapi.IndicesPutTemplateRequest{
		Name: templateName,
		Body: bytes.NewReader(data),
	}
	_, eserr := DoRequest[map[string]interface{}](req)
	if eserr != nil {
		return eserr.OriginErr
	}
	return nil
}

func CreateEsIndex(properties map[string]interface{}, indexName string) *EsErrorResult {
	var payload map[string]interface{} = map[string]interface{}{
		// "template": map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   DefaultEsConfig.NumberOfShards,
			"number_of_replicas": DefaultEsConfig.NumberOfReplicas,
		},
		"mappings": map[string]interface{}{
			// "dynamic":    false,
			"properties": properties,
			"dynamic_templates": []map[string]interface{}{
				{"strings": map[string]interface{}{"match_mapping_type": "string", "match": "*", "mapping": map[string]interface{}{"type": "keyword"}}},
			},
		},
		// },
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return NewEsError(fmt.Errorf("%s error: %s", indexName, err.Error()))
	}
	logs.Info(string(data))
	// Set up the request object.
	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  bytes.NewReader(data),
	}
	_, eserr := DoRequest[map[string]interface{}](req)
	if eserr != nil {
		return eserr
	}
	return nil
}

func AppendFilter(condition map[string]interface{}) []map[string]interface{} {
	filter := []map[string]interface{}{}
	for key, val := range condition {
		s := fmt.Sprintf("%v", val)
		if len(strings.TrimSpace(s)) > 0 && s != "<nil>" {
			var term map[string]interface{} = map[string]interface{}{}
			if strings.Contains(key, "$IN") {
				prop := strings.ReplaceAll(key, "$IN", "")
				kind := reflect.TypeOf(val).Kind()
				if kind == reflect.Array || kind == reflect.Slice {
					term["terms"] = map[string]interface{}{prop: val}
				} else {
					term["terms"] = map[string]interface{}{prop: strings.Split(s, ",")}
				}
			} else if strings.Contains(key, "$LIKE") {
				prop := strings.ReplaceAll(key, "$LIKE", "")
				term["prefix"] = map[string]interface{}{prop: s}
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

func CreateDoc(index string, docId string, ob any) error {
	b, err := json.Marshal(ob)
	if err != nil {
		return err
	}
	req := esapi.CreateRequest{
		Index: index,
		Body:  bytes.NewReader(b),
	}
	if len(docId) > 0 {
		req.DocumentID = docId
	}
	_, eserr := DoRequest[map[string]interface{}](req)
	if eserr != nil {
		return eserr.OriginErr
	}
	return nil
}

func UpdateDoc(index string, docId string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req := esapi.UpdateRequest{
		Index:      index,
		DocumentID: docId,
		Body:       bytes.NewReader([]byte(fmt.Sprintf(`{"doc": %s}`, string(b)))),
	}
	_, eserr := DoRequest[map[string]interface{}](req)
	if eserr != nil {
		return eserr.OriginErr
	}
	return nil
}

func BulkDoc(data []byte) error {
	req := esapi.BulkRequest{
		Body: bytes.NewReader([]byte(data)),
	}
	_, err := DoRequest[map[string]interface{}](req)
	if err != nil {
		return err.OriginErr
	}
	return nil
}

func UpdateDocByQuery(index string, filter []map[string]interface{}, script map[string]interface{}) error {
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filter,
			},
		},
		"script": script,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req := esapi.UpdateByQueryRequest{
		Index: []string{index},
		Body:  bytes.NewReader(data),
	}
	_, eserr := DoRequest[map[string]interface{}](req)
	if eserr != nil {
		return eserr.OriginErr
	}
	return nil
}

func DeleteDoc(index string, docId string) error {
	req := esapi.DeleteRequest{
		Index:      index,
		DocumentID: docId,
	}
	_, err := DoRequest[map[string]interface{}](req)
	if err != nil {
		return err.OriginErr
	}
	return nil
}

func DeleteByQuery(index string, filter []map[string]interface{}) error {
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filter,
			},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req := esapi.DeleteByQueryRequest{
		Index: []string{index},
		Body:  bytes.NewReader(data),
	}
	_, eserr := DoRequest[map[string]interface{}](req)
	if eserr != nil {
		return eserr.OriginErr
	}
	return nil
}

func FilterSearch[T any](index string, q Query) (int64, []T, error) {
	var total int64
	body := map[string]interface{}{
		"from": q.From,
		"size": q.Size,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": q.Filter,
			},
		},
	}
	if len(q.Includes) > 0 {
		body["_source"] = q.Includes
	}
	if len(q.Sort) > 0 {
		body["sort"] = q.Sort
	}
	var result []T
	data, err := json.Marshal(body)
	if err != nil {
		return total, result, err
	}
	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  bytes.NewReader(data),
	}
	r, eserr := DoRequest[EsQueryResult[T]](req)
	if eserr != nil {
		return total, result, eserr.OriginErr
	}
	if eserr == nil && len(r.Hits.Hits) > 0 {
		total = int64(r.Hits.Total.Value)
		// convert each hit to result.
		for _, hit := range r.Hits.Hits {
			d := hit.Source
			result = append(result, d)
		}
	}
	return total, result, nil
}
