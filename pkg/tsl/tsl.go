// 物模型
package tsl

import (
	"encoding/json"
	"errors"
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
		return fmt.Errorf("tsl parse functions error: %v", err)
	}
	tsl.Functions = functions
	list, err := parsePropertys(text, "properties")
	if err != nil {
		return fmt.Errorf("tsl parse properties error: %v", err)
	}
	tsl.Properties = list
	list, err = parsePropertys(text, "events")
	if err != nil {
		return fmt.Errorf("tsl parse events error: %v", err)
	}
	tsl.Events = list
	data, err := json.Marshal(tsl)
	if err != nil {
		return fmt.Errorf("tsl parse error: %v", err)
	}
	tsl.Text = string(data)
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
	Async       bool              `json:"async"`
	Inputs      []Property        `json:"inputs"`
	Output      Property          `json:"output"`
	Expands     map[string]string `json:"expands,omitempty"`
	Description string            `json:"description,omitempty"`
}

type Property interface {
	GetId() string
	GetName() string
	GetType() string
	GetExpands() map[string]string
	IsObject() (*PropertyObject, bool)
	setId(string)
	setType(string)
}

type tslProperty struct {
	Id          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Expands     map[string]string `json:"expands,omitempty"`
	Description string            `json:"description,omitempty"`
}

func (p *tslProperty) GetId() string {
	return p.Id
}
func (p *tslProperty) GetName() string {
	return p.Name
}

func (p tslProperty) GetExpands() map[string]string {
	return p.Expands
}

// is ValueTypeObject
func (p *tslProperty) IsObject() (*PropertyObject, bool) {
	return nil, false
}

func (p *tslProperty) setId(t string) {
	p.Id = t
}
func (p *tslProperty) setType(t string) {
	p.Type = t
}

type PropertyEnum struct {
	tslProperty
	Elements []EnumElement `json:"elements"`
}

func NewPropertyEnum(id, name string, elements []EnumElement) *PropertyEnum {
	p := &PropertyEnum{tslProperty: tslProperty{Id: id, Name: name}, Elements: elements}
	p.setType(p.GetType())
	return p
}
func (p *PropertyEnum) GetType() string {
	return TypeEnum
}

func (v *PropertyEnum) Valid() error {
	if len(v.Elements) == 0 {
		return errors.New("enum elements is empty")
	}
	for _, v := range v.Elements {
		if len(v.Value) == 0 {
			return errors.New("enum elements value is empty")
		}
		err := idCheck(v.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

type EnumElement struct {
	Text  string `json:"text"`
	Value string `json:"value"`
}

type PropertyInt struct {
	tslProperty
	Unit string `json:"unit"`
}

func NewPropertyInt(id, name string) *PropertyInt {
	p := &PropertyInt{tslProperty: tslProperty{Id: id, Name: name}}
	p.setType(p.GetType())
	return p
}

func (p *PropertyInt) GetType() string {
	return TypeInt
}

type PropertyLong struct {
	PropertyInt
}

func NewPropertyLong(id, name string) *PropertyLong {
	p := &PropertyLong{PropertyInt: *NewPropertyInt(id, name)}
	p.setType(p.GetType())
	return p
}

func (p *PropertyLong) GetType() string {
	return TypeLong
}

type PropertyBool struct {
	tslProperty
	TrueText   string `json:"trueText"`
	TrueValue  string `json:"trueValue"`
	FalseText  string `json:"falseText"`
	FalseValue string `json:"falseValue"`
}

func NewPropertyBool(id, name string) *PropertyBool {
	p := &PropertyBool{tslProperty: tslProperty{Id: id, Name: name}}
	p.setType(p.GetType())
	return p
}
func (p *PropertyBool) GetType() string {
	return TypeBool
}

type PropertyString struct {
	tslProperty
	Max int32 `json:"max"`
}

func NewPropertyString(id, name string) *PropertyString {
	p := &PropertyString{tslProperty: tslProperty{Id: id, Name: name}}
	p.setType(p.GetType())
	return p
}

func (p *PropertyString) GetType() string {
	return TypeString
}

type PropertyDate struct {
	tslProperty
	Format string `json:"format"`
}

func NewPropertyDate(id, name string) *PropertyDate {
	p := &PropertyDate{tslProperty: tslProperty{Id: id, Name: name}}
	p.setType(p.GetType())
	return p
}

func (p *PropertyDate) GetType() string {
	return TypeDate
}

type PropertyPassword struct {
	PropertyString
}

func NewPropertyPassword(id, name string) *PropertyPassword {
	p := &PropertyPassword{PropertyString: *NewPropertyString(id, name)}
	p.setType(p.GetType())
	return p
}

func (p *PropertyPassword) GetType() string {
	return TypePassword
}

type PropertyFile struct {
	tslProperty
	BodyType string `json:"bodyType"` // url, base64
}

func NewPropertyFile(id, name string) *PropertyFile {
	p := &PropertyFile{tslProperty: tslProperty{Id: id, Name: name}}
	p.setType(p.GetType())
	return p
}
func (p *PropertyFile) GetType() string {
	return TypeFile
}

type PropertyFloat struct {
	tslProperty
	Scale int32  `json:"scale"`
	Unit  string `json:"unit"`
}

func NewPropertyFloat(id, name string) *PropertyFloat {
	p := &PropertyFloat{tslProperty: tslProperty{Id: id, Name: name}}
	p.setType(p.GetType())
	return p
}
func (p *PropertyFloat) GetType() string {
	return TypeFloat
}

type PropertyDouble struct {
	PropertyFloat
}

func NewPropertyDouble(id, name string) *PropertyDouble {
	p := &PropertyDouble{PropertyFloat: *NewPropertyFloat(id, name)}
	p.setType(p.GetType())
	return p
}
func (p *PropertyDouble) GetType() string {
	return TypeDouble
}

type PropertyObject struct {
	tslProperty
	Properties []Property `json:"properties"`
}

func NewPropertyObject(id, name string, properties []Property) *PropertyObject {
	p := &PropertyObject{tslProperty: tslProperty{Id: id, Name: name}, Properties: properties}
	p.setType(p.GetType())
	return p
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

func parseFunctions(d string) ([]Function, error) {
	list := []Function{}
	functions := gjson.Get(d, "functions")
	if !functions.Exists() || functions.Value() == nil {
		return list, nil
	}
	var err1 error
	functions.ForEach(func(key, value gjson.Result) bool {
		p := Function{}
		inputs, err := parsePropertys(value.Raw, "inputs")
		if err != nil {
			err1 = err
			return false
		}
		p.Inputs = inputs
		outputval := value.Get("output")
		if len(outputval.Map()) > 0 {
			output, err := parseProperty(outputval, false)
			output.setId("result")
			if err != nil {
				err1 = err
				return false
			}
			p.Output = output
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
		p.Description = value.Get("description").String()
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

func parsePropertys(d string, key string) ([]Property, error) {
	res := gjson.Get(d, key)
	var list []Property
	if !res.Exists() || res.Value() == nil {
		return list, nil
	}
	var err error
	res.ForEach(func(key, value gjson.Result) bool {
		var property Property
		property, err = parseProperty(value, true)
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

func parseProperty(value gjson.Result, idMustNotNull bool) (Property, error) {
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
	property.setType(property.GetType())
	switch d1 := property.(type) {
	case *PropertyObject:
		d1.Id = value.Get("id").String()
		d1.Name = value.Get("name").String()
		d1.Expands = map[string]string{}
		d1.Description = value.Get("description").String()
		value.Get("expands").ForEach(func(key, value gjson.Result) bool {
			d1.Expands[key.String()] = value.String()
			return true
		})
		var properties []Property
		value.Get("properties").ForEach(func(key, value gjson.Result) bool {
			p, e := parseProperty(value, idMustNotNull)
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
	if idMustNotNull {
		if len(strings.TrimSpace(property.GetId())) == 0 {
			err = fmt.Errorf("id of tslProperty must be persent")
			return nil, err
		}
		err = idCheck(property.GetId())
		if err != nil {
			return nil, err
		}
	}
	return property, err
}

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
