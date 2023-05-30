package orm

import (
	"fmt"
	"go-iot/pkg/core/es"
	"reflect"
	"strings"
)

// 1 is attr
// 2 is tag
var supportTag = map[string]int{
	"-":    1,
	"pk":   1,
	"size": 2,
	"type": 2,
}

const (
	defaultStructTagName  = "orm"
	defaultStructTagDelim = ";"
)

// RegisterModel register models
func RegisterModel(models ...interface{}) {
	if err := register(es.Prefix, models...); err != nil {
		panic(err)
	}
}

// RegisterModelWithPrefix register models with a prefix
func RegisterModelWithPrefix(prefix string, models ...interface{}) {
	if err := register(prefix, models...); err != nil {
		panic(err)
	}
}

// get reflect.Type name with package path.
func getFullName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

func GetIndexName(model interface{}) string {
	val := reflect.ValueOf(model)
	typ := reflect.Indirect(val).Type()
	if val.Kind() != reflect.Ptr {
		err := fmt.Errorf("<es RegisterModel> cannot use non-ptr model struct `%s`", getFullName(typ))
		panic(err)
	}
	// For this case:
	// u := &User{}
	// registerModel(&u)
	if typ.Kind() == reflect.Ptr {
		err := fmt.Errorf("<es RegisterModel> only allow ptr model struct, it looks you use two reference to the struct `%s`", typ)
		panic(err)
	}
	indexName := es.Prefix + getIndexName(val)
	return indexName
}

// getIndexName get struct table name.
// If the struct implement the IndexName, then get the result as tablename
// else use the struct name which will apply snakeString.
func getIndexName(val reflect.Value) string {
	if fun := val.MethodByName("IndexName"); fun.IsValid() {
		vals := fun.Call([]reflect.Value{})
		// has return and the first val is string
		if len(vals) > 0 && vals[0].Kind() == reflect.String {
			return strings.ToLower(vals[0].String())
		}
	}
	return strings.ToLower(reflect.Indirect(val).Type().Name())
}

// Abc => abc, ABC => aBC
func FirstLower(str string) string {
	return strings.ToLower(str[:1]) + str[1:]
}

// abc => Abc, aBc => ABc
func FirstUpper(str string) string {
	return strings.ToUpper(str[:1]) + str[1:]
}

type modelInfo struct {
	pkg        string
	name       string
	fullName   string
	indexName  string
	pkkey      string
	fieldNames []string
}

func (mc *modelInfo) getPKValue(md interface{}) string {
	val := mc.getFieldValue(md, mc.pkkey)
	docId := fmt.Sprintf("%v", val)
	return docId
}
func (mc *modelInfo) setPKValue(md interface{}, value interface{}) {
	mc.setFieldValue(md, mc.pkkey, value)
}

func (mc *modelInfo) getFieldValue(md interface{}, fieldName string) interface{} {
	r := reflect.ValueOf(md)
	if r.Kind() == reflect.Pointer {
		r = reflect.Indirect(r)
	}
	val := r.FieldByName(FirstUpper(fieldName)).Interface()
	return val
}

func (mc *modelInfo) setFieldValue(md interface{}, fieldName string, value interface{}) {
	r := reflect.ValueOf(md)
	if r.Kind() == reflect.Pointer {
		r = reflect.Indirect(r)
	}
	f := r.FieldByName(FirstUpper(fieldName))
	switch f.Type().Kind() {
	case reflect.Int64:
		f.SetInt(value.(int64))
	case reflect.String:
		f.SetString(value.(string))
	}
}

func (mc *modelInfo) addField(fieldName string) {
	mc.fieldNames = append(mc.fieldNames, fieldName)
}

var defaultmodelCache = &modelCache{
	cache:           map[string]*modelInfo{},
	cacheByFullName: map[string]*modelInfo{},
}

type modelCache struct {
	cache           map[string]*modelInfo
	cacheByFullName map[string]*modelInfo
}

// get model info by full name
func (mc *modelCache) getByFullName(name string) (mi *modelInfo, ok bool) {
	mi, ok = mc.cacheByFullName[name]
	return
}

func (mc *modelCache) getByMd(md interface{}) (*modelInfo, bool) {
	val := reflect.ValueOf(md)
	ind := reflect.Indirect(val)
	typ := ind.Type()
	name := getFullName(typ)
	return mc.getByFullName(name)
}

// set model info to collection
func (mc *modelCache) set(indexName string, mi *modelInfo) *modelInfo {
	mii := mc.cache[indexName]
	mc.cache[indexName] = mi
	mc.cacheByFullName[mi.fullName] = mi

	return mii
}

// new model info
func newModelInfo(val reflect.Value) (mi *modelInfo) {
	mi = &modelInfo{}
	ind := reflect.Indirect(val)
	mi.name = ind.Type().Name()
	mi.pkg = ind.Type().PkgPath()
	mi.fullName = getFullName(ind.Type())
	// mi.indexName = getIndexName(val)
	return
}

func register(prefix string, models ...interface{}) (err error) {
	for _, model := range models {
		val := reflect.ValueOf(model)
		typ := reflect.Indirect(val).Type()

		if val.Kind() != reflect.Ptr {
			err = fmt.Errorf("<es RegisterModel> cannot use non-ptr model struct `%s`", getFullName(typ))
			return
		}
		// For this case:
		// u := &User{}
		// registerModel(&u)
		if typ.Kind() == reflect.Ptr {
			err = fmt.Errorf("<es RegisterModel> only allow ptr model struct, it looks you use two reference to the struct `%s`", typ)
			return
		}
		if val.Elem().Kind() == reflect.Slice {
			val = reflect.New(val.Elem().Type().Elem())
		}
		indexName := getIndexName(val)
		if prefix != "" {
			indexName = prefix + indexName
		}
		mi := newModelInfo(val)
		mi.indexName = indexName
		defaultmodelCache.set(indexName, mi)
		ind := reflect.Indirect(val)
		var properties map[string]interface{} = map[string]interface{}{}
		for i := 0; i < ind.NumField(); i++ {
			sf := ind.Type().Field(i)
			// if the field is unexported skip
			if sf.PkgPath != "" {
				continue
			}

			attrs, tags := parseStructTag(sf.Tag.Get(defaultStructTagName))
			if _, ok := attrs["-"]; ok {
				continue
			}
			if _, ok := attrs["pk"]; ok {
				mi.pkkey = sf.Name
			} else {
				mi.pkkey = "Id"
			}
			mi.addField(sf.Name)
			fieldName := FirstLower(sf.Name)
			typ := tags["type"]
			if len(typ) > 0 {
				if typ == "date" {
					properties[fieldName] = es.Property{Type: "date", Format: es.DefaultDateFormat}
				} else {
					properties[fieldName] = es.Property{Type: typ}
				}
			} else {
				switch sf.Type.Kind() {
				case reflect.Bool:
					properties[fieldName] = es.Property{Type: "boolean"}
				case reflect.Float32:
					properties[fieldName] = es.Property{Type: "float"}
				case reflect.Float64:
					properties[fieldName] = es.Property{Type: "double"}
				case reflect.Int:
				case reflect.Int16:
				case reflect.Int32:
					properties[fieldName] = es.Property{Type: "integer"}
				case reflect.Int8:
					properties[fieldName] = es.Property{Type: "byte"}
				case reflect.Int64:
					properties[fieldName] = es.Property{Type: "long"}
				case reflect.Map:
					properties[fieldName] = map[string]interface{}{"type": "object", "properties": map[string]es.Property{"a": {Type: "keyword"}}}
				case reflect.Slice:
					properties[fieldName] = map[string]interface{}{"type": "nested"}
				default:
					switch sf.Type.String() {
					case "time.Time":
						properties[fieldName] = es.Property{Type: "date", Format: es.DefaultDateFormat}
					default:
						properties[fieldName] = es.Property{Type: "keyword"}
					}
				}
			}
		}
		err := es.CreateEsTemplate(properties, indexName, indexName+"-template", "10ms")
		if err != nil {
			return err
		}

	}
	return
}

// parse struct tag string
func parseStructTag(data string) (attrs map[string]bool, tags map[string]string) {
	attrs = make(map[string]bool)
	tags = make(map[string]string)
	for _, v := range strings.Split(data, defaultStructTagDelim) {
		if v == "" {
			continue
		}
		v = strings.TrimSpace(v)
		if t := strings.ToLower(v); supportTag[t] == 1 {
			attrs[t] = true
		} else if i := strings.Index(v, "("); i > 0 && strings.Index(v, ")") == len(v)-1 {
			name := t[:i]
			if supportTag[name] == 2 {
				v = v[i+1 : len(v)-1]
				tags[name] = v
			}
		}
	}
	return
}
