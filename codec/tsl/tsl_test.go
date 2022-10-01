package tsl_test

import (
	"encoding/json"
	"go-iot/codec/tsl"
	"log"
	"testing"
)

const text = `
{
  "events": [],
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
      "id": "voltage",
      "name": "电压",
      "valueType": {
        "type": "double",
        "scale": 2,
        "unit": "volt"
      },
      "expands": {
        "readOnly": "true"
      }
    },
    {
      "id": "power",
      "name": "功率",
      "valueType": {
        "type": "double",
        "scale": 2,
        "unit": "watt"
      },
      "expands": {
        "readOnly": "true"
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
	err := json.Unmarshal([]byte(text), &d)
	if err != nil {
		t.Error(err)
	}
	for _, e := range d.Events {
		log.Println(e.Id)
		log.Println(e.Name)
		for _, p := range e.Properties {
			log.Println(p.GetValueType())
		}
	}
	for _, e := range d.Functions {
		log.Println(e.Id)
		for _, p := range e.Inputs {
			log.Println(p.GetValueType())
		}
		log.Println(e.Outputs.GetValueType())
	}
	for _, e := range d.Properties {
		log.Println(e.GetValueType())
	}
}
