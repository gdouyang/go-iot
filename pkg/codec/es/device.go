package es

import (
	"bytes"
	"encoding/json"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

const deviceIndex string = "goiot-device"

func PageDevice[T any](from int, size int, condition map[string]interface{}) (int64, []T, error) {
	var total int64
	var result []T
	filter := AppendFilter(condition)
	body := map[string]interface{}{
		"from": from,
		"size": size,
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
		return 0, result, err
	}
	req := esapi.SearchRequest{
		Index: []string{deviceIndex},
		Body:  bytes.NewReader(data),
	}
	r, err := DoRequest[EsQueryResult[T]](req)
	if err == nil && len(r.Hits.Hits) > 0 {
		total = int64(r.Hits.Total.Value)
		// convert each hit to result.
		for _, hit := range r.Hits.Hits {
			d := hit.Source
			result = append(result, d)
		}
	}
	return total, result, err
}

func AddDevice(deviceId string, text string) error {
	req := esapi.CreateRequest{
		Index:      deviceIndex,
		DocumentID: deviceId,
		Body:       bytes.NewReader([]byte(text)),
	}
	_, err := DoRequest[map[string]interface{}](req)
	return err
}

func UpdateDevice(deviceId string, text string) error {
	req := esapi.UpdateRequest{
		Index:      deviceIndex,
		DocumentID: deviceId,
		Body:       bytes.NewReader([]byte(text)),
	}
	_, err := DoRequest[map[string]interface{}](req)
	return err
}

func DeleteDevice(deviceId string) error {
	req := esapi.DeleteRequest{
		Index:      deviceIndex,
		DocumentID: deviceId,
	}
	_, err := DoRequest[map[string]interface{}](req)
	return err
}
