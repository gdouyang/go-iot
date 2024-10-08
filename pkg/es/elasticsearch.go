// elasticsearch操作
package es

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"reflect"
	"strings"

	logs "go-iot/pkg/logger"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/tidwall/gjson"
)

var Err404 = errors.New("404")

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

/**
 * 创建索引模板
 */
func CreateEsTemplate(properties map[string]any, indexPattern string, templateName string, refresh_interval string) error {
	settings := map[string]any{
		"number_of_shards":               DefaultEsConfig.NumberOfShards,
		"number_of_replicas":             DefaultEsConfig.NumberOfReplicas,
		"index.mapping.ignore_malformed": true, // 忽略类型值错误
	}
	if len(refresh_interval) > 0 {
		settings["refresh_interval"] = refresh_interval
	}
	var payload map[string]any = map[string]any{
		"index_patterns": []string{indexPattern},
		"order":          0,
		"settings":       settings,
		"mappings": map[string]any{
			// "dynamic":    false,
			"properties": properties,
			"dynamic_templates": []map[string]any{
				{"strings": map[string]any{"match_mapping_type": "string", "match": "*", "mapping": map[string]any{"type": "keyword"}}},
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s error: %s", templateName, err.Error())
	}
	logs.Infof(string(data))
	// Set up the request object.
	req := esapi.IndicesPutTemplateRequest{
		Name: templateName,
		Body: bytes.NewReader(data),
	}
	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	if resp.IsError {
		return fmt.Errorf("%s", resp.Data)
	}
	return nil
}

/**
 * 创建索引
 */
func CreateEsIndex(properties map[string]any, indexName string) error {
	var payload map[string]any = map[string]any{
		// "template": map[string]any{
		"settings": map[string]any{
			"number_of_shards":   DefaultEsConfig.NumberOfShards,
			"number_of_replicas": DefaultEsConfig.NumberOfReplicas,
		},
		"mappings": map[string]any{
			// "dynamic":    false,
			"properties": properties,
			"dynamic_templates": []map[string]any{
				{"strings": map[string]any{"match_mapping_type": "string", "match": "*", "mapping": map[string]any{"type": "keyword"}}},
			},
		},
		// },
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s error: %s", indexName, err.Error())
	}
	logs.Infof(string(data))
	// Set up the request object.
	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  bytes.NewReader(data),
	}
	resp, eserr := DoRequest(req)
	if eserr != nil {
		return eserr
	}
	if resp.IsError {
		return fmt.Errorf("%s error: %s", indexName, resp.Data)
	}
	return nil
}

/**
 * 删除索引
 */
func DeleteIndex(index ...string) error {
	var IgnoreUnavailable bool = true
	req := esapi.IndicesDeleteRequest{
		Index:             index,
		IgnoreUnavailable: &IgnoreUnavailable,
	}
	resp, eserr := DoRequest(req)
	if eserr != nil {
		return eserr
	}
	if resp.IsError && !resp.Is404() {
		return errors.New(resp.Data)
	}
	return nil
}

/**
 * 创建文档
 */
func CreateDoc(index string, docId string, ob any) error {
	b, err := json.Marshal(ob)
	if err != nil {
		return err
	}
	if logs.IsDebug() {
		logs.Debugf("==> %s create %s", index, string(b))
	}
	req := esapi.CreateRequest{
		Index:   index,
		Body:    bytes.NewReader(b),
		Refresh: "true",
	}
	if len(docId) > 0 {
		req.DocumentID = docId
	}
	resp, eserr := DoRequest(req)
	if eserr != nil {
		return eserr
	}
	if resp.IsError {
		return fmt.Errorf("%s error: %s", index, resp.Data)
	}
	return nil
}

/**
 * 更新文档
 */
func UpdateDoc(index string, docId string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if logs.IsDebug() {
		logs.Debugf("==> %s update %s", index, string(b))
	}
	req := esapi.UpdateRequest{
		Index:      index,
		DocumentID: docId,
		Body:       bytes.NewReader([]byte(fmt.Sprintf(`{"doc": %s}`, string(b)))),
		Refresh:    "true",
	}
	resp, eserr := DoRequest(req)
	if eserr != nil {
		return eserr
	}
	if resp.IsError {
		return fmt.Errorf("%s error: %s", index, resp.Data)
	}
	return nil
}

/**
 * 批量创建文档
 */
func BulkDoc(data []byte) error {
	req := esapi.BulkRequest{
		Body: bytes.NewReader([]byte(data)),
		// Refresh: "true",
	}
	resp, eserr := DoRequest(req)
	if eserr != nil {
		return eserr
	}
	if resp.IsError {
		return fmt.Errorf("error: %s", resp.Data)
	}
	return nil
}

/**
 * 更新文档
 */
func UpdateDocByQuery(index string, filter []map[string]any, script map[string]any) error {
	body := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filter,
			},
		},
		"script": script,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if logs.IsDebug() {
		logs.Debugf("==> %s update_by_query %s", index, string(data))
	}
	refresh := true
	req := esapi.UpdateByQueryRequest{
		Index:     []string{index},
		Body:      bytes.NewReader(data),
		Conflicts: "proceed",
		Refresh:   &refresh,
	}
	resp, eserr := DoRequest(req)
	if eserr != nil {
		return eserr
	}
	if resp.IsError {
		return fmt.Errorf("%s error: %s", index, resp.Data)
	}
	return nil
}

/**
 * 删除文档
 */
func DeleteDoc(index string, docId string) error {
	req := esapi.DeleteRequest{
		Index:      index,
		DocumentID: docId,
		Refresh:    "true",
	}
	resp, eserr := DoRequest(req)
	if eserr != nil {
		return eserr
	}
	if resp.IsError && !resp.Is404() {
		return fmt.Errorf("%s error: %s", index, resp.Data)
	}
	return nil
}

/**
 * 删除文档
 */
func DeleteByQuery(index string, filter []map[string]any) error {
	body := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filter,
			},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if logs.IsDebug() {
		logs.Debugf("==> %s delete_by_query", index, string(data))
	}
	refresh := true
	req := esapi.DeleteByQueryRequest{
		Index:     []string{index},
		Body:      bytes.NewReader(data),
		Conflicts: "proceed",
		Refresh:   &refresh,
	}
	resp, eserr := DoRequest(req)
	if eserr != nil {
		return eserr
	}
	if resp.IsError && !resp.Is404() {
		return fmt.Errorf("%s error: %s", index, resp.Data)
	}
	return nil
}

/**
 * 统计
 */
func FilterCount(q Query, indexs ...string) (int64, error) {
	body := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": q.Filter,
			},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	if logs.IsDebug() {
		logs.Debugf("==> %s %s %s", strings.Join(indexs, ","), "count", string(data))
	}
	ignoreUnavailable := true
	req := esapi.CountRequest{
		Index:             indexs,
		Body:              bytes.NewReader(data),
		IgnoreUnavailable: &ignoreUnavailable,
	}
	res, eserr := DoRequest(req)
	if eserr != nil {
		return 0, eserr
	}
	if res.IsError && !res.Is404() {
		return 0, fmt.Errorf("error: %s", res.Data)
	}

	str := res.Data
	if res.Is404() {
		str = `{"count": 0}`
	}
	if logs.IsDebug() {
		logs.Debugf("<== %s %s %s", strings.Join(indexs, ","), "count", str)
	}
	total := gjson.Get(str, "count")
	return total.Int(), nil
}

/**
 * 搜索, 使用filter查询, indexs必填
 */
func FilterSearch(q Query, indexs ...string) (*SearchResponse, error) {
	if len(indexs) == 0 {
		return nil, errors.New("indexs must be persent")
	}
	body := map[string]any{
		"from": q.From,
		"size": q.Size,
		"query": map[string]any{
			"bool": map[string]any{
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
	if len(q.SearchAfter) > 0 {
		body["from"] = 0
		body["search_after"] = q.SearchAfter
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	if logs.IsDebug() {
		logs.Debugf("==> %s %s %s", strings.Join(indexs, ","), "search", string(data))
	}
	ignoreUnavailable := true
	req := esapi.SearchRequest{
		Index:             indexs,
		Body:              bytes.NewReader(data),
		IgnoreUnavailable: &ignoreUnavailable, // 不存在的索引不报错
	}
	res, eserr := DoRequest(req)
	if eserr != nil {
		return nil, eserr
	}
	if res.IsError && !res.Is404() {
		return nil, fmt.Errorf("error: %s", res.Data)
	}

	str := res.Data
	if res.Is404() {
		str = `{"hits": {"total":{"value": 0}, "hits": []}}`
	}
	if logs.IsDebug() {
		logs.Debugf("<== %s %s %s", strings.Join(indexs, ","), "search", str)
	}
	var resp SearchResponse
	total := gjson.Get(str, "hits.total.value")
	resp.Total = total.Int()
	hits := gjson.Get(str, "hits.hits")
	buf := bytes.Buffer{}
	buf.WriteString("[")
	if hits.IsArray() {
		array := hits.Array()
		length := len(array)
		for idx, v := range array {
			_source := gjson.Get(v.Raw, "_source")
			buf.WriteString(_source.Raw)
			if idx == 0 {
				resp.FirstSource = []byte(_source.Raw)
			}
			if idx < length-1 {
				buf.WriteString(",")
			} else {
				sort := gjson.Get(v.Raw, "sort")
				if sort.Exists() {
					lastSort := []any{}
					err = json.Unmarshal([]byte(sort.Raw), &lastSort)
					if err != nil {
						return nil, err
					}
					resp.LastSort = lastSort
				}
			}
		}
	}
	buf.WriteString("]")
	resp.Sources = buf.Bytes()
	return &resp, nil
}

/**
 * 搜索, 追加条件
 */
func AppendFilter(condition []core.SearchTerm) []map[string]any {
	filter := []map[string]any{}
	for _, _term := range condition {
		if _term.Value == nil {
			continue
		}
		key := _term.Key
		value := _term.Value
		var term map[string]any = map[string]any{}
		switch _term.Oper {
		case core.IN:
			kind := reflect.TypeOf(value).Kind()
			if kind == reflect.Array || kind == reflect.Slice {
				term["terms"] = map[string]any{key: value}
			} else {
				s := fmt.Sprintf("%v", value)
				term["terms"] = map[string]any{key: strings.Split(s, ",")}
			}
		case core.LIKE:
			term["prefix"] = map[string]any{key: value}
		case core.GT:
			term["range"] = map[string]any{key: map[string]any{"gt": value}}
		case core.GTE:
			term["range"] = map[string]any{key: map[string]any{"gte": value}}
		case core.LT:
			term["range"] = map[string]any{key: map[string]any{"lt": value}}
		case core.LTE:
			term["range"] = map[string]any{key: map[string]any{"lte": value}}
		case core.BTW:
			s := fmt.Sprintf("%v", value)
			vals := strings.Split(s, ",")
			if len(vals) < 2 {
				continue
			}
			term["range"] = map[string]any{key: map[string]any{
				"gte": vals[0],
				"lte": vals[1],
			}}
		case core.NEQ:
			term["bool"] = map[string]any{"must_not": []map[string]any{{"term": map[string]any{key: value}}}}
		case core.NOTNULL:
			term["exists"] = map[string]any{"field": key}
		default:
			term["term"] = map[string]any{key: value}
		}
		filter = append(filter, term)
	}
	return filter
}

/**
 * 执行es请求
 */
func DoRequest(s esDoFunc) (EsResponse, error) {
	es, err := getEsClient()
	if err != nil {
		logs.Errorf("error creating the client: %v", err)
	}
	var esResp EsResponse
	// Perform the request with the client.
	res, err := s.Do(context.Background(), es)
	if err != nil {
		logs.Errorf("error getting response: %v", err)
		return esResp, err
	}
	defer res.Body.Close()

	var buf bytes.Buffer
	buf.ReadFrom(res.Body)
	esResp.Data = buf.String()
	esResp.StatusCode = res.StatusCode
	esResp.IsError = res.IsError()
	return esResp, nil
}
