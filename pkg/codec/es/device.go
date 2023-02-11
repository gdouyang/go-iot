package es

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func PageDevice[T []any](from int, size int, condition map[string]interface{}) (T, error) {
	var result T
	filter := []map[string]interface{}{
		// {"term": map[string]interface{}{
		// 	tsl.PropertyDeviceId: param.DeviceId,
		// },
		// }
	}
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
		return result, err
	}
	req := esapi.SearchRequest{
		Index: []string{"goiot-device"},
		Body:  bytes.NewReader(data),
	}
	result, err = DoRequest[T](req)
	return result, err
}

func AddDevice(deviceId string, text string) error {
	req := esapi.CreateRequest{
		Index:      "goiot-device",
		DocumentID: deviceId,
		Body:       bytes.NewReader([]byte(text)),
	}
	_, err := DoRequest[map[string]interface{}](req)
	return err
}

func UpdateDevice(deviceId string, text string) error {
	req := esapi.UpdateRequest{
		Index:      "goiot-device",
		DocumentID: deviceId,
		Body:       bytes.NewReader([]byte(text)),
	}
	_, err := DoRequest[map[string]interface{}](req)
	return err
}

func DeleteDevice(deviceId string) error {
	req := esapi.DeleteRequest{
		Index:      "goiot-device",
		DocumentID: deviceId,
	}
	_, err := DoRequest[map[string]interface{}](req)
	return err
}
