package tsl

import (
	"encoding/json"
	"fmt"
	"regexp"
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
	// TypeArray    = "array"

	PropertyDeviceId = "deviceId"
)

type TslData struct {
	Functions  []TslFunction `json:"functions"`
	Events     []TslProperty `json:"events"`
	Properties []TslProperty `json:"properties"`
	Text       string        `json:"-"`
}

func NewTslData() *TslData {
	return &TslData{}
}

func (tsl *TslData) FromJson(text string) error {
	err := json.Unmarshal([]byte(text), tsl)
	if err != nil {
		return fmt.Errorf("tsl parse error: %v", err)
	}
	{
		var idMap map[string]bool = map[string]bool{}
		for _, v := range tsl.Functions {
			if _, ok := idMap[v.Id]; ok {
				return fmt.Errorf("tsl parse error: functions is repeat [%s]", v.Id)
			} else {
				idMap[v.Id] = true
			}
		}
	}
	{
		var idMap map[string]bool = map[string]bool{}
		for _, v := range tsl.Properties {
			if obj, ok := v.IsObject(); ok {
				if len(obj.Properties) == 0 {
					return fmt.Errorf("tsl parse error: properties [%s] must have value", v.Id)
				}
			}
			if _, ok := idMap[v.Id]; ok {
				return fmt.Errorf("tsl parse error: properties is repeat [%s]", v.Id)
			} else {
				idMap[v.Id] = true
			}
		}
	}
	{
		var idMap map[string]bool = map[string]bool{}
		for _, v := range tsl.Events {
			if obj, ok := v.IsObject(); ok {
				if len(obj.Properties) == 0 {
					return fmt.Errorf("tsl parse error: events [%s] must have value", v.Id)
				}
			}
			if _, ok := idMap[v.Id]; ok {
				return fmt.Errorf("tsl parse error: events is repeat [%s]", v.Id)
			} else {
				idMap[v.Id] = true
			}
		}
	}
	tsl.Text = text
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

func (tsl *TslData) EventsMap() map[string]TslProperty {
	tslF := map[string]TslProperty{}
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
	Async   bool              `json:"async"`
	Inputs  []TslProperty     `json:"inputs"`
	Outputs TslProperty       `json:"output"`
	Expands map[string]string `json:"expands,omitempty"`
}

func (p *TslFunction) UnmarshalJSON(d []byte) error {
	var alias struct {
		Id   string `json:"id"`
		Name string `json:"name"`
		// 是否异步调用
		Async   bool              `json:"async"`
		Inputs  []TslProperty     `json:"inputs,omitempty"`
		Outputs TslProperty       `json:"output,omitempty"`
		Expands map[string]string `json:"expands,omitempty"`
	}
	err := json.Unmarshal(d, &alias)
	if err != nil {
		return fmt.Errorf("function of tsl has error: [%s], data: %s", err.Error(), string(d))
	}
	err = idCheck(alias.Id)
	if err != nil {
		return err
	}
	p.Id = alias.Id
	p.Name = alias.Name
	p.Async = alias.Async
	p.Inputs = alias.Inputs
	p.Outputs = alias.Outputs
	p.Expands = alias.Expands
	return nil
}

type TslProperty struct {
	Id        string            `json:"id"`
	Name      string            `json:"name"`
	ValueType interface{}       `json:"valueType"`
	Expands   map[string]string `json:"expands,omitempty"`
	Type      string            `json:"-"`
}

func (p *TslProperty) UnmarshalJSON(d []byte) error {
	var alias struct {
		Id        string                 `json:"id"`
		Name      string                 `json:"name"`
		ValueType map[string]interface{} `json:"valueType"`
		Expands   map[string]string      `json:"expands,omitempty"`
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
	err = idCheck(p.Id)
	if err != nil {
		return err
	}
	p.Name = alias.Name
	p.Expands = alias.Expands
	p.Type = fmt.Sprintf("%v", t)
	switch p.Type {
	case TypeEnum:
		valueType := ValueTypeEnum{}
		err = convert(alias.ValueType, &valueType)
		// if err == nil && valueType.Valid() != nil {
		// 	return valueType.Valid()
		// }
		p.ValueType = valueType
	case TypeInt:
		valueType := ValueTypeInt{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	case TypeLong:
		valueType := ValueTypeInt{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	case TypeString:
		valueType := ValueTypeString{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	case TypeBool:
		valueType := ValueTypeBool{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	case TypePassword:
		valueType := ValueTypePassword{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	case TypeFloat:
		valueType := ValueTypeFloat{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	case TypeDouble:
		valueType := ValueTypeFloat{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	case TypeDate:
		valueType := ValueTypeString{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	case TypeFile:
		valueType := ValueTypeFile{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	// case TypeArray:
	// 	valueType := ValueTypeArray{}
	// 	err = convert(alias.ValueType, &valueType)
	// 	p.ValueType = valueType
	case TypeObject:
		valueType := ValueTypeObject{}
		err = convert(alias.ValueType, &valueType)
		p.ValueType = valueType
	default:
		return fmt.Errorf("valueType %v is not support", t)
	}
	return err
}

// is ValueTypeObject
func (p *TslProperty) IsObject() (*ValueTypeObject, bool) {
	switch d := (p.ValueType).(type) {
	case ValueTypeObject:
		return &d, true
	}
	return nil, false
}

type ValueTypeEnum struct {
	Type     string             `json:"type"`
	Elements []ValueTypeEnumEle `json:"elements"`
}

// func (v *ValueTypeEnum) Valid() error {
// 	if len(v.Elements) == 0 {
// 		return errors.New("enum elements is empty")
// 	}
// 	for _, v := range v.Elements {
// 		if len(v.Value) == 0 {
// 			return errors.New("enum elements value is empty")
// 		}
// 		err := idCheck(v.Value)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

type ValueTypeEnumEle struct {
	Text  string `json:"text"`
	Value string `json:"value"`
}

type ValueTypeInt struct {
	Type string `json:"type"`
	Max  int32  `json:"max"`
	Min  int32  `json:"min"`
}

type ValueTypeBool struct {
	Type       string `json:"type"`
	TrueText   string `json:"trueText"`
	TrueValue  string `json:"trueValue"`
	FalseText  string `json:"falseText"`
	FalseValue string `json:"falseValue"`
}

type ValueTypeString struct {
	Type string `json:"type"`
	Max  int32  `json:"max"`
	Min  int32  `json:"min"`
}

type ValueTypePassword struct {
	Type string `json:"type"`
	Max  int32  `json:"max"`
	Min  int32  `json:"min"`
}

type ValueTypeFile struct {
	Type     string `json:"type"`
	BodyType string `json:"bodyType"` // url, base64
}

type ValueTypeFloat struct {
	Type  string `json:"type"`
	Scale int32  `json:"scale"`
	Unit  string `json:"unit"`
	Max   int32  `json:"max"`
	Min   int32  `json:"min"`
}

type ValueTypeObject struct {
	Type       string        `json:"type"`
	Properties []TslProperty `json:"properties"`
}

func (p *ValueTypeObject) PropertiesMap() map[string]TslProperty {
	tslP := map[string]TslProperty{}
	for _, p := range p.Properties {
		tslP[p.Id] = p
	}
	return tslP
}

type ValueTypeArray struct {
	Type        string      `json:"type"`
	ElementType TslProperty `json:"elementType"`
}

func convert(data map[string]interface{}, v any) error {
	str, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(str, v)
}

func idCheck(id string) error {
	matched, _ := regexp.Match("^[0-9a-zA-Z_\\-]+$", []byte(id))
	if !matched {
		return fmt.Errorf("id [%s] is invalid, must be alphabet,number,underscores", id)
	}
	return nil
}
