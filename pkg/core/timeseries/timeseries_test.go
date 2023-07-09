package timeseries_test

import (
	"encoding/json"
	"go-iot/pkg/core"
	"go-iot/pkg/core/store"
	"go-iot/pkg/core/timeseries"
	"go-iot/pkg/core/tsl"
	"testing"

	logs "go-iot/pkg/logger"
)

const text = `
{
  "events": [
		{
      "name": "fire_alarm",
      "id": "fire_alarm",
      "expands": {
        "level": "ordinary"
      },
			"valueType": {
        "type": "object",
        "properties": [
          {
            "valueType": {
              "type": "float"
            },
            "name": "lnt",
            "id": "lnt",
            "expands": {}
          },
          {
            "valueType": {
              "type": "float"
            },
            "name": "lat",
            "id": "lat",
            "expands": {}
          }
        ]
      }
    }
	],
  "properties": [
    {
      "id": "light",
      "name": "亮度",
      "valueType": {
        "type": "int",
        "unit": ""
      },
      "expands": {
        "readOnly": "true"
      }
    },
    {
      "id": "current",
      "name": "电流",
      "valueType": {
        "type": "double",
        "scale": 2,
        "unit": "milliAmpere"
      },
      "expands": {
        "readOnly": "true"
      }
    },
    {
      "id": "curr0abc",
      "name": "电流",
      "valueType": {
        "type": "double",
        "scale": 2,
        "unit": "milliAmpere"
      },
      "expands": {
        "readOnly": "true"
      }
    },
    {
      "id": "obj",
      "name": "obj",
      "type": "object",
      "expands": {
        "readOnly": "true",
        "level": null
      },
      "description": null,
      "valueType": {
				"type": "object",
        "properties": [
          {
            "id": "name",
            "name": "name",
            "expands": {
              "readOnly": null,
              "level": null
            },
            "description": "test",
            "valueType": {
              "expands": {
                "maxLength": "32"
              },
              "type": "string"
            }
          },
          {
            "id": "curr0abc",
            "name": "电流",
            "valueType": {
              "type": "double",
              "scale": 2,
              "unit": "milliAmpere"
            },
            "expands": {
              "readOnly": "true"
            }
          }
        ]
      }
    }
  ],
  "functions": []
}
`

func TestTdengine(t *testing.T) {
	ts := timeseries.TdengineTimeSeries{}
	core.RegDeviceStore(store.NewMockDeviceStore())
	product, err := core.NewProduct("test", map[string]string{}, "tdengien", text)
	if err != nil {
		t.Error(err)
	}
	core.PutProduct(product)
	device := core.NewDevice("1234", product.Id, 0)
	core.PutDevice(device)

	d := tsl.NewTslData()
	d.FromJson(text)
	product.TslData = d

	ts.Del(product)

	ts.PublishModel(product, *d)

	err = ts.SaveProperties(product, map[string]any{"deviceId": device.Id, "current": 10})
	if err != nil {
		logs.Errorf(err.Error())
	}
	{
		query := core.TimeDataSearchRequest{}
		query.DeviceId = device.Id
		query.Condition = []core.SearchTerm{}
		res, err := ts.QueryProperty(product, query)
		if err != nil {
			logs.Errorf(err.Error())
		}
		d, _ := json.Marshal(res)
		println(string(d))
	}
	err = ts.SaveEvents(product, "fire_alarm", map[string]any{"deviceId": device.Id, "lnt": 90, "lat": 100})
	if err != nil {
		logs.Errorf(err.Error())
	}
	{
		query := core.TimeDataSearchRequest{}
		query.DeviceId = device.Id
		query.Condition = []core.SearchTerm{}
		res, err := ts.QueryEvent(product, "fire_alarm", query)
		if err != nil {
			logs.Errorf(err.Error())
		}
		d, _ := json.Marshal(res)
		println(string(d))
	}
	ts.SaveLogs(product, core.LogData{DeviceId: device.Id, Content: `{"state":"online"}`, Type: "online"})
	{
		query := core.TimeDataSearchRequest{}
		query.DeviceId = device.Id
		query.Condition = []core.SearchTerm{}
		res, err := ts.QueryLogs(product, query)
		if err != nil {
			logs.Errorf(err.Error())
		}
		d, _ := json.Marshal(res)
		println(string(d))
	}
	{
		query := core.TimeDataSearchRequest{}
		query.DeviceId = device.Id
		query.Condition = []core.SearchTerm{
			{Key: "createTime", Value: []string{"2023-06-11 19:45:00.000", "2023-06-11 19:46:00.000"}, Oper: core.IN},
			{Key: "createTime", Value: []string{"2023-06-11 19:45:00", "2023-06-11 19:46:00"}, Oper: core.BTW},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.GT},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.GTE},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.LT},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.LTE},
			{Key: "obj.name", Value: "test", Oper: core.LIKE},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.NEQ},
		}
		ts.QueryProperty(product, query)
	}
	{
		query := core.TimeDataSearchRequest{}
		query.DeviceId = device.Id
		query.Condition = []core.SearchTerm{
			{Key: "createTime", Value: []string{"2023-06-11 19:45:00.000", "2023-06-11 19:46:00.000"}, Oper: core.IN},
			{Key: "createTime", Value: []string{"2023-06-11 19:45:00", "2023-06-11 19:46:00"}, Oper: core.BTW},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.GT},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.GTE},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.LT},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.LTE},
			{Key: "createTime", Value: "2023-06-11 19:45:00.000", Oper: core.NEQ},
		}
		ts.QueryLogs(product, query)
	}
}
