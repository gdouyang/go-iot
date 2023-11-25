package tsl

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
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
	Functions  []Function `json:"functions"`
	Events     []Property `json:"events"`
	Properties []Property `json:"properties"`
	Text       string     `json:"-"`
}

func NewTslData() *TslData {
	return &TslData{}
}

func (tsl *TslData) FromJson(text string) error {
	functions, err := parseFunctions(text)
	if err != nil {
		return fmt.Errorf("tsl parse error: %v", err)
	}
	tsl.Functions = functions
	list, err := parsePropertys(text, "properties")
	if err != nil {
		return fmt.Errorf("tsl parse error: %v", err)
	}
	tsl.Properties = list
	list, err = parsePropertys(text, "events")
	if err != nil {
		return fmt.Errorf("tsl parse error: %v", err)
	}
	tsl.Events = list
	tsl.Text = text
	return nil
}

func (tsl *TslData) PropertiesMap() map[string]Property {
	tslP := map[string]Property{}
	for _, p := range tsl.Properties {
		tslP[p.GetId()] = p
	}
	return tslP
}

func (tsl *TslData) FunctionsMap() map[string]Function {
	tslF := map[string]Function{}
	for _, p := range tsl.Functions {
		tslF[p.Id] = p
	}
	return tslF
}

func (tsl *TslData) EventsMap() map[string]Property {
	tslF := map[string]Property{}
	for _, p := range tsl.Events {
		tslF[p.GetId()] = p
	}
	return tslF
}

type Function struct {
	// function id
	Id   string `json:"id"`
	Name string `json:"name"`
	// 是否异步调用
	Async   bool              `json:"async"`
	Inputs  []Property        `json:"inputs"`
	Outputs Property          `json:"output"`
	Expands map[string]string `json:"expands,omitempty"`
}

func parseFunctions(d string) ([]Function, error) {
	list := []Function{}
	var err1 error
	gjson.Get(d, "functions").ForEach(func(key, value gjson.Result) bool {
		p := Function{}
		inputs, err := parsePropertys(value.Raw, "inputs")
		if err != nil {
			err1 = err
			return false
		}
		p.Inputs = inputs
		outputval := gjson.Get(d, "output")
		if len(outputval.Map()) > 0 {
			output, err := parseProperty(gjson.Get(value.Raw, "output"))
			if err != nil {
				err1 = err
				return false
			}
			p.Outputs = output
		}
		p.Id = value.Get("id").String()
		err = idCheck(p.Id)
		if err != nil {
			err1 = err
			return false
		}
		p.Name = value.Get("name").String()
		p.Async = value.Get("async").Bool()
		p.Expands = map[string]string{}
		expands := value.Get("expands")
		if len(expands.Map()) > 0 {
			err = json.Unmarshal([]byte(value.Get("expands").Raw), &p.Expands)
			if err != nil {
				err1 = fmt.Errorf("function has error: [%s], data: %s", err.Error(), string(d))
				return false
			}
		}
		list = append(list, p)
		return true
	})
	{
		var idMap map[string]bool = map[string]bool{}
		for _, v := range list {
			if _, ok := idMap[v.Id]; ok {
				return nil, fmt.Errorf("function is repeat [%s]", v.Id)
			} else {
				idMap[v.Id] = true
			}
		}
	}
	if err1 != nil {
		return nil, err1
	}
	return list, nil
}

type Property interface {
	GetId() string
	GetName() string
	GetType() string
	GetExpands() map[string]string
	IsObject() (*PropertyObject, bool)
}

type TslProperty struct {
	Id      string            `json:"id"`
	Name    string            `json:"name"`
	Expands map[string]string `json:"expands,omitempty"`
}

func (p *TslProperty) GetId() string {
	return p.Id
}
func (p *TslProperty) GetName() string {
	return p.Name
}

func (p TslProperty) GetExpands() map[string]string {
	return p.Expands
}

// is ValueTypeObject
func (p *TslProperty) IsObject() (*PropertyObject, bool) {
	return nil, false
}

func parsePropertys(d string, key string) ([]Property, error) {
	res := gjson.Get(d, key)
	var list []Property
	var err error
	res.ForEach(func(key, value gjson.Result) bool {
		var property Property
		property, err = parseProperty(value)
		if err != nil {
			return false
		}
		list = append(list, property)
		return true
	})
	{
		var idMap map[string]bool = map[string]bool{}
		for _, v := range list {
			if obj, ok := v.IsObject(); ok {
				if len(obj.Properties) == 0 {
					return list, fmt.Errorf("%s [%s] must have properties", key, v.GetId())
				}
			}
			if _, ok := idMap[v.GetId()]; ok {
				return list, fmt.Errorf("%s is repeat [%s]", key, v.GetId())
			} else {
				idMap[v.GetId()] = true
			}
		}
	}

	return list, err
}

func parseProperty(value gjson.Result) (Property, error) {
	var err error
	ptype := gjson.Get(value.Raw, "type")
	var property Property
	typeName := ptype.String()
	if len(typeName) == 0 {
		err = fmt.Errorf("type is not exist: %s", value.Raw)
		return nil, err
	}
	switch typeName {
	case TypeEnum:
		property = &PropertyEnum{}
		// if err == nil && valueType.Valid() != nil {
		// 	return valueType.Valid()
		// }
	case TypeInt:
		property = &PropertyInt{}
	case TypeLong:
		property = &PropertyLong{}
	case TypeString:
		property = &PropertyString{}
	case TypeBool:
		property = &PropertyBool{}
	case TypePassword:
		property = &PropertyPassword{}
	case TypeFloat:
		property = &PropertyFloat{}
	case TypeDouble:
		property = &PropertyDouble{}
	case TypeDate:
		property = &PropertyDate{}
	case TypeFile:
		property = &PropertyFile{}
	case TypeObject:
		property = &PropertyObject{}
	// case TypeArray:
	// 	valueType := ValueTypeArray{}
	// 	err = convert(alias.ValueType, &valueType)
	// 	p.ValueType = valueType
	default:
		err = fmt.Errorf("type %v is not support", typeName)
		return nil, err
	}
	switch d1 := property.(type) {
	case *PropertyObject:
		d1.Id = value.Get("id").String()
		d1.Name = value.Get("name").String()
		d1.Expands = map[string]string{}
		value.Get("expands").ForEach(func(key, value gjson.Result) bool {
			d1.Expands[key.String()] = value.String()
			return true
		})
		var properties []Property
		value.Get("properties").ForEach(func(key, value gjson.Result) bool {
			p, e := parseProperty(value)
			if e != nil {
				err = e
				return false
			}
			properties = append(properties, p)
			return true
		})
		d1.Properties = properties
	default:
		err = convert(value.Raw, property)
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(property.GetId())) == 0 {
		err = fmt.Errorf("id of tslProperty must be persent")
		return nil, err
	}
	err = idCheck(property.GetId())
	if err != nil {
		return nil, err
	}
	return property, err
}

type PropertyEnum struct {
	TslProperty
	Elements []EnumElement `json:"elements"`
}

func (p *PropertyEnum) GetType() string {
	return TypeEnum
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

type EnumElement struct {
	Text  string `json:"text"`
	Value string `json:"value"`
}

type PropertyInt struct {
	TslProperty
	Max int32 `json:"max"`
	Min int32 `json:"min"`
}

func (p *PropertyInt) GetType() string {
	return TypeInt
}

type PropertyLong struct {
	TslProperty
	Max int32 `json:"max"`
	Min int32 `json:"min"`
}

func (p *PropertyLong) GetType() string {
	return TypeLong
}

type PropertyBool struct {
	TslProperty
	TrueText   string `json:"trueText"`
	TrueValue  string `json:"trueValue"`
	FalseText  string `json:"falseText"`
	FalseValue string `json:"falseValue"`
}

func (p *PropertyBool) GetType() string {
	return TypeBool
}

type PropertyString struct {
	TslProperty
	Max int32 `json:"max"`
	Min int32 `json:"min"`
}

func (p *PropertyString) GetType() string {
	return TypeString
}

type PropertyDate struct {
	TslProperty
}

func (p *PropertyDate) GetType() string {
	return TypeDate
}

type PropertyPassword struct {
	TslProperty
	Max int32 `json:"max"`
	Min int32 `json:"min"`
}

func (p *PropertyPassword) GetType() string {
	return TypePassword
}

type PropertyFile struct {
	TslProperty
	BodyType string `json:"bodyType"` // url, base64
}

func (p *PropertyFile) GetType() string {
	return TypeFile
}

type PropertyFloat struct {
	TslProperty
	Scale int32  `json:"scale"`
	Unit  string `json:"unit"`
	Max   int32  `json:"max"`
	Min   int32  `json:"min"`
}

func (p *PropertyFloat) GetType() string {
	return TypeFloat
}

type PropertyDouble struct {
	TslProperty
	Scale int32  `json:"scale"`
	Unit  string `json:"unit"`
	Max   int32  `json:"max"`
	Min   int32  `json:"min"`
}

func (p *PropertyDouble) GetType() string {
	return TypeDouble
}

type PropertyObject struct {
	TslProperty
	Properties []Property `json:"properties"`
}

func (p *PropertyObject) GetType() string {
	return TypeObject
}

func (p *PropertyObject) IsObject() (*PropertyObject, bool) {
	return p, true
}

func (p *PropertyObject) PropertiesMap() map[string]Property {
	tslP := map[string]Property{}
	for _, p := range p.Properties {
		tslP[p.GetId()] = p
	}
	return tslP
}

// type PropertyArray struct {
// 	TslProperty
// 	ElementType TslProperty `json:"elementType"`
// }

// func (p *PropertyArray) GetType() string {
// 	return TypeArray
// }

func convert(data string, v any) error {
	return json.Unmarshal([]byte(data), v)
}

func idCheck(id string) error {
	matched, _ := regexp.Match("^[0-9a-zA-Z_\\-]+$", []byte(id))
	if !matched {
		return fmt.Errorf("id [%s] is invalid, must be alphabet,number,underscores", id)
	}
	return nil
}
