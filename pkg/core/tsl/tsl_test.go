package tsl_test

import (
	"fmt"
	"go-iot/pkg/core/tsl"
	"log"
	"testing"
)

const text = `
{
  "events": [
		{
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
          },
          {
            "valueType": {
              "type": "int"
            },
            "name": "point",
            "id": "point",
            "expands": {}
          },
          {
            "valueType": {
              "expands": {},
              "type": "string"
            },
            "name": "b_name",
            "id": "b_name",
            "expands": {}
          }
        ]
      },
      "name": "fire_alarm",
      "id": "fire_alarm",
      "expands": {
        "level": "ordinary"
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
      "id": "obj",
      "name": "obj",
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
          }
        ],
        "type": "object"
      }
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
          "valueType": {
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
        }
      ]
    },
    {
      "id": "dimming",
      "name": "调光",
      "async": false,
      "output": {},
      "inputs": [
        {
          "id": "bright",
          "name": "亮度",
          "valueType": {
            "type": "int"
          }
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
          "valueType": {
            "type": "string"
          }
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
	for _, e := range d.Events {
		log.Println(e.Id)
		log.Println(e.Name)
		log.Println(e.ValueType)
	}
	for _, e := range d.Functions {
		log.Println(e.Id)
		for _, p := range e.Inputs {
			log.Println(p.ValueType)
		}
		log.Println(e.Outputs.ValueType)
	}
	for _, e := range d.Properties {
		log.Println(e.ValueType)
	}
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
