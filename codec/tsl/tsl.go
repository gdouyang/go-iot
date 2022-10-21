package tsl

import (
	"encoding/json"
)

const (
	TypeEnum   = "enum"
	TypeInt    = "int"
	TypeString = "string"
	TypeBool   = "bool"
	TypeFloat  = "float"
	TypeDouble = "double"
	TypeDate   = "date"

	PropertyDeviceId = "deviceId"
)

type TslData struct {
	Functions  []TslFunction `json:"functions"`
	Events     []TslEvent    `json:"events"`
	Properties []TslProperty `json:"properties"`
}

func (tsl *TslData) FromJson(text string) error {
	err := json.Unmarshal([]byte(text), tsl)
	return err
}

func (tsl *TslData) PropertiesMap() map[string]TslProperty {
	tslP := map[string]TslProperty{}
	for _, p := range tsl.Properties {
		tslP[p.Id] = p
	}
	return tslP
}

func (tsl *TslData) FunctionsMap() map[string]TslFunction {
	tslF := map[string]TslFunction{}
	for _, p := range tsl.Functions {
		tslF[p.Id] = p
	}
	return tslF
}

type TslFunction struct {
	// function id
	Id   string `json:"id"`
	Name string `json:"name"`
	// 是否异步调用
	Async   bool          `json:"async"`
	Inputs  []TslProperty `json:"inputs"`
	Outputs TslProperty   `json:"output"`
}

type TslEvent struct {
	Id         string        `json:"id"`
	Name       string        `json:"name"`
	Properties []TslProperty `json:"properties"`
}

type TslProperty struct {
	Id        string                 `json:"id"`
	Name      string                 `json:"name"`
	ValueType map[string]interface{} `json:"valueType"`
	Expands   map[string]interface{} `json:"expands"`
}

func (p *TslProperty) GetValueType() interface{} {
	t, ok := p.ValueType["type"]
	if !ok {
		return p.ValueType
	}
	switch t.(string) {
	case TypeEnum:
		data, _ := json.Marshal(p.ValueType)
		valueType := ValueTypeEnum{}
		json.Unmarshal(data, &valueType)
		return valueType
	case TypeInt:
		data, _ := json.Marshal(p.ValueType)
		valueType := ValueTypeInt{}
		json.Unmarshal(data, &valueType)
		return valueType
	case TypeString:
		data, _ := json.Marshal(p.ValueType)
		valueType := ValueTypeString{}
		json.Unmarshal(data, &valueType)
		return valueType
	case TypeFloat:
		data, _ := json.Marshal(p.ValueType)
		valueType := ValueTypeFloat{}
		json.Unmarshal(data, &valueType)
		return valueType
	case TypeDouble:
		data, _ := json.Marshal(p.ValueType)
		valueType := ValueTypeFloat{}
		json.Unmarshal(data, &valueType)
		return valueType
	}
	return p.ValueType
}

type ValueTypeEnum struct {
	Type     string             `json:"type"`
	Elements []ValueTypeEnumEle `json:"elements"`
}

type ValueTypeEnumEle struct {
	Text  string `json:"text"`
	Value string `json:"value"`
	Id    string `json:"id"`
}

type ValueTypeInt struct {
	Type string `json:"type"`
	Max  int32  `json:"max"`
	Min  int32  `json:"min"`
}

type ValueTypeString struct {
	Type string `json:"type"`
	Max  int32  `json:"max"`
	Min  int32  `json:"min"`
}

type ValueTypeFloat struct {
	Type  string `json:"type"`
	Scale int32  `json:"scale"`
	Unit  string `json:"unit"`
	Max   int32  `json:"max"`
	Min   int32  `json:"min"`
}
