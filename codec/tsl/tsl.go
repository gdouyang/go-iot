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
	TypeObject = "object"

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
		valueType := ValueTypeEnum{}
		valueType.convert(p.ValueType)
		return valueType
	case TypeInt:
		valueType := ValueTypeInt{}
		valueType.convert(p.ValueType)
		return valueType
	case TypeString:
		valueType := ValueTypeString{}
		valueType.convert(p.ValueType)
		return valueType
	case TypeFloat:
		valueType := ValueTypeFloat{}
		valueType.convert(p.ValueType)
		return valueType
	case TypeDouble:
		valueType := ValueTypeFloat{}
		valueType.convert(p.ValueType)
		return valueType
	}
	return p.ValueType
}

type ValueTypeEnum struct {
	Type     string             `json:"type"`
	Elements []ValueTypeEnumEle `json:"elements"`
}

func (v *ValueTypeEnum) convert(data map[string]interface{}) error {
	str, _ := json.Marshal(data)
	return json.Unmarshal(str, v)
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

func (v *ValueTypeInt) convert(data map[string]interface{}) error {
	str, _ := json.Marshal(data)
	return json.Unmarshal(str, v)
}

type ValueTypeString struct {
	Type string `json:"type"`
	Max  int32  `json:"max"`
	Min  int32  `json:"min"`
}

func (v *ValueTypeString) convert(data map[string]interface{}) error {
	str, _ := json.Marshal(data)
	return json.Unmarshal(str, v)
}

type ValueTypeFloat struct {
	Type  string `json:"type"`
	Scale int32  `json:"scale"`
	Unit  string `json:"unit"`
	Max   int32  `json:"max"`
	Min   int32  `json:"min"`
}

func (v *ValueTypeFloat) convert(data map[string]interface{}) error {
	str, _ := json.Marshal(data)
	return json.Unmarshal(str, v)
}
