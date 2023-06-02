package es_test

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestXxx(t *testing.T) {
	var str = `{
	"took": 1,
	"timed_out": false,
	"_shards": {
		"total": 1,
		"successful": 1,
		"skipped": 0,
		"failed": 0
	},
	"hits": {
		"total": {
			"value": 1,
			"relation": "eq"
		},
		"max_score": null,
		"hits": [
			{
				"_index": "goiot-device",
				"_id": "1111",
				"_score": null,
				"_source": {
					"deviceType": "device",
					"productId": "mqtt",
					"createTime": "2023-05-28 13:26:27",
					"createId": 1,
					"name": "1111-沙发上",
					"id": "1111",
					"state": "offline"
				},
				"sort": [
					"2023-05-28 13:26:27"
				]
			}
		]
	}
}`
	var resp map[string]interface{}
	json.Unmarshal([]byte(str), &resp)
	fmt.Println(resp)
}
