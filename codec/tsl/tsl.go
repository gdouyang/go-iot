package tsl

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	TypeEnum     = "enum" // 枚举类型
	TypeInt      = "int"
	TypeLong     = "long"
	TypeString   = "string"
	TypeBool     = "bool"
	TypeFloat    = "float"
	TypeDouble   = "double"
	TypeDate     = "date"
	TypePassword = "password"
	TypeFile     = "file"
	TypeObject   = "object"
	TypeArray    = "array"

	PropertyDeviceId = "deviceId"
)

type TslData struct {
	Functions  []TslFunction `json:"functions"`
	Events     []TslEvent    `json:"events"`
	Properties []TslProperty `json:"properties"`
}

func NewTslData() *TslData {
	return &TslData{}
}

func (tsl *TslData) FromJson(text string) error {
	err := json.Unmarshal([]byte(text), tsl)
	if err != nil {
		return fmt.Errorf("tsl parse error: %v", err)
	}
	return nil
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

func (tsl *TslData) EventsMap() map[string]TslEvent {
	tslF := map[string]TslEvent{}
	for _, p := range tsl.Events {
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

func (p *TslFunction) UnmarshalJSON(d []byte) error {
	var alias struct {
		Id   string `json:"id"`
		Name string `json:"name"`
		// 是否异步调用
		Async   bool          `json:"async"`
		Inputs  []TslProperty `json:"inputs,omitempty"`
		Outputs TslProperty   `json:"output,omitempty"`
	}
	err := json.Unmarshal(d, &alias)
	if err != nil {
		return fmt.Errorf("function of tsl has error: [%s], data: %s", err.Error(), string(d))
	}
	p.Id = alias.Id
	p.Name = alias.Name
	p.Async = alias.Async
	p.Inputs = alias.Inputs
	p.Outputs = alias.Outputs
	return nil
}

type TslEvent struct {
	Id         string        `json:"id"`
	Name       string        `json:"name"`
	Properties []TslProperty `json:"properties"`
}

func (e *TslEvent) UnmarshalJSON(d []byte) error {
	var alias struct {
		Id         string        `json:"id"`
		Name       string        `json:"name"`
		Properties []TslProperty `json:"properties"`
	}
	err := json.Unmarshal(d, &alias)
	if err != nil {
		return fmt.Errorf("event of tsl has error: [%s], data: %s", err.Error(), string(d))
	}
	e.Id = alias.Id
	e.Name = alias.Name
	e.Properties = alias.Properties
	return nil
}

func (e *TslEvent) PropertiesMap() map[string]TslProperty {
	tslP := map[string]TslProperty{}
	for _, p := range e.Properties {
		tslP[p.Id] = p
	}
	return tslP
}

type TslProperty struct {
	Id        string                 `json:"id"`
	Name      string                 `json:"name"`
	ValueType interface{}            `json:"valueType"`
	Expands   map[string]interface{} `json:"expands,omitempty"`
	Type      string                 `json:"-"`
}

func (p *TslProperty) UnmarshalJSON(d []byte) error {
	var alias struct {
		Id        string                 `json:"id"`
		Name      string                 `json:"name"`
		ValueType map[string]interface{} `json:"valueType"`
		Expands   map[string]interface{} `json:"expands,omitempty"`
		Type      string                 `json:"-"`
	}
	err := json.Unmarshal(d, &alias)
	if err != nil {
		return err
	}
	t, ok := alias.ValueType["type"]
	if !ok {
		return nil
	}
	if len(strings.TrimSpace(alias.Id)) == 0 {
		return fmt.Errorf("id of tslProperty must be persent")
	}
	p.Id = alias.Id
	p.Name = alias.Name
	p.Expands = alias.Expands
	p.Type = fmt.Sprintf("%v", t)
	var valueType IValueType
	switch p.Type {
	case TypeEnum:
		valueType = &ValueTypeEnum{}
	case TypeInt:
		valueType = &ValueTypeInt{}
	case TypeLong:
		valueType = &ValueTypeInt{}
	case TypeString:
		valueType = &ValueTypeString{}
	case TypeBool:
		valueType = &ValueTypeBool{}
	case TypePassword:
		valueType = &ValueTypePassword{}
	case TypeFloat:
		valueType = &ValueTypeFloat{}
	case TypeDouble:
		valueType = &ValueTypeFloat{}
	case TypeFile:
		valueType = &ValueTypeFile{}
	case TypeArray:
		valueType = &ValueTypeArray{}
	case TypeObject:
		valueType = &ValueTypeObject{}
	default:
		return fmt.Errorf("valueType %v is invalid", t)
	}
	err = valueType.convert(alias.ValueType)
	p.ValueType = valueType
	return err
}

type IValueType interface {
	convert(data map[string]interface{}) error
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
	Id    string `json:"id"`
	Text  string `json:"text"`
	Value string `json:"value"`
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

type ValueTypeBool struct {
	Type       string `json:"type"`
	TrueText   string `json:"trueText"`
	TrueValue  string `json:"trueValue"`
	FalseText  string `json:"falseText"`
	FalseValue string `json:"falseValue"`
}

func (v *ValueTypeBool) convert(data map[string]interface{}) error {
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

type ValueTypePassword struct {
	Type string `json:"type"`
	Max  int32  `json:"max"`
	Min  int32  `json:"min"`
}

func (v *ValueTypePassword) convert(data map[string]interface{}) error {
	str, _ := json.Marshal(data)
	return json.Unmarshal(str, v)
}

type ValueTypeFile struct {
	Type     string `json:"type"`
	BodyType string `json:"bodyType"` // url, base64
}

func (v *ValueTypeFile) convert(data map[string]interface{}) error {
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

type ValueTypeObject struct {
	Type       string        `json:"type"`
	Properties []TslProperty `json:"properties"`
}

func (v *ValueTypeObject) convert(data map[string]interface{}) error {
	str, _ := json.Marshal(data)
	return json.Unmarshal(str, v)
}

type ValueTypeArray struct {
	Type        string      `json:"type"`
	ElementType TslProperty `json:"elementType"`
}

func (v *ValueTypeArray) convert(data map[string]interface{}) error {
	str, _ := json.Marshal(data)
	return json.Unmarshal(str, v)
}
