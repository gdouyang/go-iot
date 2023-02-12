package es

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-iot/pkg/core/boot"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func init() {
	boot.AddStartLinstener(func() {
		err := initMapping()
		if err != nil {
			panic(err)
		}
	})
}

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
	r, eserr := DoRequest[EsQueryResult[T]](req)
	if eserr == nil && len(r.Hits.Hits) > 0 {
		total = int64(r.Hits.Total.Value)
		// convert each hit to result.
		for _, hit := range r.Hits.Hits {
			d := hit.Source
			result = append(result, d)
		}
	}
	return total, result, err
}

func AddDevice(deviceId string, data map[string]interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req := esapi.CreateRequest{
		Index:      deviceIndex,
		DocumentID: deviceId,
		Body:       bytes.NewReader(b),
	}
	_, eserr := DoRequest[map[string]interface{}](req)
	if eserr != nil {
		return eserr.OriginErr
	}
	return nil
}

func UpdateDevice(deviceId string, data map[string]interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req := esapi.UpdateRequest{
		Index:      deviceIndex,
		DocumentID: deviceId,
		Body:       bytes.NewReader([]byte(fmt.Sprintf(`{"doc": %s}`, string(b)))),
	}
	_, eserr := DoRequest[map[string]interface{}](req)
	if eserr != nil {
		return eserr.OriginErr
	}
	return nil
}

func DeleteDevice(deviceId string) error {
	req := esapi.DeleteRequest{
		Index:      deviceIndex,
		DocumentID: deviceId,
	}
	_, err := DoRequest[map[string]interface{}](req)
	if err != nil {
		return err.OriginErr
	}
	return nil
}

func UpdateOnlineStatusList(ids []string, state string) error {
	var data []byte
	for _, id := range ids {
		o := `{ "update" : { "_index" : "` + deviceIndex + `", "_id": "` + id + `" } }` + "\n" + fmt.Sprintf(`{"doc": {"state": "%s"}}`, state) + "\n"
		data = append(data, []byte(o)...)
	}
	req := esapi.BulkRequest{
		Body: bytes.NewReader([]byte(data)),
	}
	_, err := DoRequest[map[string]interface{}](req)
	if err != nil {
		return err.OriginErr
	}
	return nil
}

func initMapping() error {
	var properties map[string]interface{} = map[string]interface{}{}
	properties["id"] = EsType{Type: "keyword"}
	properties["productId"] = EsType{Type: "keyword"}
	properties["state"] = EsType{Type: "keyword"}
	properties["metaconfig"] = EsType{Type: "object"}
	properties["tag"] = EsType{Type: "object"}
	properties["createId"] = EsType{Type: "long"}
	properties["createTime"] = EsType{Type: "date", Format: DefaultDateFormat}
	properties["name"] = EsType{Type: "text"}
	properties["desc"] = EsType{Type: "text"}
	err := CreateEsIndex(properties, deviceIndex)
	if err != nil {
		if err.Error.Type == "resource_already_exists_exception" {
			return nil
		}
		return err.OriginErr
	}
	return nil
}
