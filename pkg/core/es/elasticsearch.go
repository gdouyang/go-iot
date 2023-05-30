package es

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/beego/beego/v2/core/logs"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type esDoFunc interface {
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

func DoRequest[T any](s esDoFunc) (T, *ErrorResponse) {
	var result T
	es, err := getEsClient()
	if err != nil {
		logs.Error("Error creating the client: %s", err)
	}
	// Perform the request with the client.
	res, err := s.Do(context.Background(), es)
	if err != nil {
		logs.Error("Error getting response: %s", err)
		return result, &ErrorResponse{Info: &ErrorInfo{Reason: err.Error()}}
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return result, nil
	}

	if res.IsError() {
		// Deserialize the response into a map.
		var eserr *ErrorResponse
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
		return result, &ErrorResponse{Info: &ErrorInfo{Reason: err.Error()}}
	}
	return result, nil
}

func CreateEsTemplate(properties map[string]interface{}, indexPattern string, templateName string, refresh_interval string) error {
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
		return errors.New(eserr.Info.Phase)
	}
	return nil
}

func CreateEsIndex(properties map[string]interface{}, indexName string) *ErrorResponse {
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

func AppendFilter(condition []SearchTerm) []map[string]interface{} {
	filter := []map[string]interface{}{}
	for _, val := range condition {
		if val.Value == nil {
			continue
		}
		key := val.Key
		var term map[string]interface{} = map[string]interface{}{}
		switch val.Oper {
		case IN:
			kind := reflect.TypeOf(val).Kind()
			if kind == reflect.Array || kind == reflect.Slice {
				term["terms"] = map[string]interface{}{key: val}
			} else {
				s := fmt.Sprintf("%v", val.Value)
				term["terms"] = map[string]interface{}{key: strings.Split(s, ",")}
			}
		case LIKE:
			term["prefix"] = map[string]interface{}{key: val.Value}
		case GT:
			term["gt"] = map[string]interface{}{key: val.Value}
		case GTE:
			term["gte"] = map[string]interface{}{key: val.Value}
		case LT:
			term["lt"] = map[string]interface{}{key: val.Value}
		case LTE:
			term["lte"] = map[string]interface{}{key: val.Value}
		case BTW:
			s := fmt.Sprintf("%v", val.Value)
			vals := strings.Split(s, ",")
			if len(vals) < 1 {
				continue
			}
			term["range"] = map[string]interface{}{key: map[string]interface{}{
				"gte": vals[0],
				"lte": vals[1],
			}}
		default:
			term["term"] = map[string]interface{}{key: val}
		}
		filter = append(filter, term)
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
		return eserr.Error()
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
		return eserr.Error()
	}
	return nil
}

func BulkDoc(data []byte) error {
	req := esapi.BulkRequest{
		Body: bytes.NewReader([]byte(data)),
	}
	_, err := DoRequest[map[string]interface{}](req)
	if err != nil {
		return err.Error()
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
		return eserr.Error()
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
		return err.Error()
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
		return eserr.Error()
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
	r, eserr := DoRequest[SearchResponse[T]](req)
	if eserr != nil {
		return total, result, eserr.Error()
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
