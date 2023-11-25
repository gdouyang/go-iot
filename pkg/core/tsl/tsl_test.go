package tsl_test

import (
	"fmt"
	"go-iot/pkg/core/tsl"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

const text = `
{
  "events": [
		{
      "type": "object",
      "name": "fire_alarm",
      "id": "fire_alarm",
      "expands": {
        "level": "ordinary"
      },
      "properties": [
        {
          "type": "float",
          "name": "lnt",
          "id": "lnt",
          "expands": {}
        },
        {
          "type": "double",
          "name": "lat",
          "id": "lat",
          "expands": {}
        },
        {
          "type": "int",
          "name": "point",
          "id": "point",
          "expands": {}
        },
        {
          "type": "string",
          "name": "b_name",
          "id": "b_name",
          "expands": {}
        }
      ]
    }
	],
  "properties": [
    {
      "id": "light",
      "name": "亮度",
      "type": "int",
      "unit": "",
      "expands": {
        "readOnly": "true"
      }
    },
    {
      "id": "current",
      "name": "电流",
      "type": "double",
      "scale": 2,
      "unit": "milliAmpere",
      "expands": {
        "readOnly": "true"
      }
    },
    {
      "id": "obj",
      "name": "obj",
      "expands": {
        "readOnly": "true",
        "level": null
      },
      "description": null,
      "type": "object",
      "properties": [
        {
          "id": "name",
          "name": "名称",
          "type": "string",
          "description": "test",
          "maxLength": "32",
          "expands": {
            "readOnly": null,
            "level": null
          }
        }
      ]
    }
  ],
  "functions": [
    {
      "id": "switching",
      "name": "开关",
      "async": false,
      "output": {},
      "inputs": [
        {
          "id": "status",
          "name": "状态",
          "type": "enum",
          "elements": [
            {
              "text": "开灯",
              "value": "on",
              "id": "0"
            },
            {
              "id": "2",
              "value": "off",
              "text": "关灯"
            }
          ]
        }
      ]
    },
    {
      "id": "dimming",
      "name": "调光",
      "async": true,
      "output": {},
      "inputs": [
        {
          "id": "bright",
          "name": "亮度",
          "type": "int"
        }
      ]
    },
    {
      "id": "strategy",
      "name": "策略",
      "async": false,
      "output": {},
      "inputs": [
        {
          "id": "strategy",
          "name": "策略",
          "type": "string"
        }
      ]
    },
    {
      "id": "timing",
      "name": "校时",
      "async": false,
      "output": {},
      "inputs": []
    }
  ],
  "tags": []
}
`

func TestTsl(t *testing.T) {
	d := tsl.TslData{}
	err := d.FromJson(text)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 1, len(d.Events), "Events size wrong")
	e := d.Events[0]
	assert.Equal(t, "fire_alarm", e.GetId())
	assert.Equal(t, "fire_alarm", e.GetName())
	assert.Equal(t, "object", e.GetType())
	obj, is := e.IsObject()
	assert.True(t, is, "Events not object")
	assert.Equal(t, 4, len(obj.Properties))
	assert.IsType(t, &tsl.PropertyFloat{}, obj.Properties[0])
	assert.Equal(t, "lnt", obj.Properties[0].GetId())
	assert.Equal(t, "lnt", obj.Properties[0].GetName())

	assert.IsType(t, &tsl.PropertyDouble{}, obj.Properties[1])
	assert.Equal(t, "lat", obj.Properties[1].GetId())
	assert.Equal(t, "lat", obj.Properties[1].GetName())

	assert.IsType(t, &tsl.PropertyInt{}, obj.Properties[2])
	assert.Equal(t, "point", obj.Properties[2].GetId())
	assert.Equal(t, "point", obj.Properties[2].GetName())

	assert.IsType(t, &tsl.PropertyString{}, obj.Properties[3])
	assert.Equal(t, "b_name", obj.Properties[3].GetId())
	assert.Equal(t, "b_name", obj.Properties[3].GetName())

	assert.Equal(t, 3, len(d.Properties))
	assert.IsType(t, &tsl.PropertyInt{}, d.Properties[0], "type error")
	assert.Equal(t, "light", d.Properties[0].GetId())
	assert.Equal(t, "亮度", d.Properties[0].GetName())

	assert.IsType(t, &tsl.PropertyDouble{}, d.Properties[1], "type error")
	assert.Equal(t, "current", d.Properties[1].GetId())
	assert.Equal(t, "电流", d.Properties[1].GetName())

	assert.IsType(t, &tsl.PropertyObject{}, d.Properties[2], "type error")
	obj, is = d.Properties[2].IsObject()
	assert.True(t, is, "Properties not object")
	assert.Equal(t, 1, len(obj.Properties))
	assert.IsType(t, &tsl.PropertyString{}, obj.Properties[0])
	assert.Equal(t, "name", obj.Properties[0].GetId())
	assert.Equal(t, "名称", obj.Properties[0].GetName())

	assert.Equal(t, 4, len(d.Functions))
	assert.Equal(t, "switching", d.Functions[0].Id)
	assert.Equal(t, "开关", d.Functions[0].Name)
	assert.Equal(t, false, d.Functions[0].Async)
	assert.Equal(t, nil, d.Functions[0].Outputs)
	assert.Equal(t, 1, len(d.Functions[0].Inputs))
	assert.IsType(t, &tsl.PropertyEnum{}, d.Functions[0].Inputs[0])

	assert.Equal(t, "dimming", d.Functions[1].Id)
	assert.Equal(t, "调光", d.Functions[1].Name)
	assert.Equal(t, true, d.Functions[1].Async)
	assert.Equal(t, nil, d.Functions[1].Outputs)
	assert.Equal(t, 1, len(d.Functions[1].Inputs))

	assert.Equal(t, "strategy", d.Functions[2].Id)
	assert.Equal(t, "策略", d.Functions[2].Name)
	assert.Equal(t, false, d.Functions[2].Async)
	assert.Equal(t, nil, d.Functions[2].Outputs)
	assert.Equal(t, 1, len(d.Functions[2].Inputs))

	assert.Equal(t, "timing", d.Functions[3].Id)
	assert.Equal(t, "校时", d.Functions[3].Name)
	assert.Equal(t, false, d.Functions[3].Async)
	assert.Equal(t, nil, d.Functions[3].Outputs)
	assert.Equal(t, 0, len(d.Functions[3].Inputs))

	s := fmt.Sprintf("%v", 1)
	log.Println(s)
	s = fmt.Sprintf("%v", 11.22)
	log.Println(s)
	s = fmt.Sprintf("%v", true)
	log.Println(s)
	s = fmt.Sprintf("%v", 100000)
	log.Println(s)
	s = fmt.Sprintf("%v", "100000ff")
	log.Println(s)
}
