package tsl

import (
	"fmt"
	"strconv"

	"github.com/beego/beego/v2/core/logs"
)

// convert value use the tsl
func ValueConvert(propertys []TslProperty, data *map[string]interface{}) {
	var propMap map[string]TslProperty = map[string]TslProperty{}
	for _, prop := range propertys {
		propMap[prop.Id] = prop
	}
	ValueConvert1(propMap, data)
}

// convert value use the tsl
func ValueConvert1(propMap map[string]TslProperty, data *map[string]interface{}) error {
	for key, value := range *data {
		if key == PropertyDeviceId {
			continue
		}
		if prop, ok := propMap[key]; !ok {
			delete(*data, key)
		} else {
			valType := fmt.Sprintf("%v", prop.ValueType["type"])
			switch valType {
			case TypeEnum:
				switch value.(type) {
				case string:
				default:
					(*data)[key] = fmt.Sprintf("%v", value)
				}
			case TypeInt:
				switch value.(type) {
				case int:
				case int16:
				case int32:
				case int64:
				default:
					s := fmt.Sprintf("%v", value)
					f, err := strconv.ParseInt(s, 10, 0)
					if err != nil {
						logs.Error(err)
					} else {
						(*data)[key] = f
					}
				}
			case TypeString:
				(*data)[key] = fmt.Sprintf("%v", value)
			case TypeFloat:
				switch value.(type) {
				case float32:
				case float64:
				default:
					s := fmt.Sprintf("%v", value)
					f, err := strconv.ParseFloat(s, 32)
					if err != nil {
						logs.Error(err)
					} else {
						(*data)[key] = f
					}
				}
			case TypeDouble:
				switch value.(type) {
				case float32:
				case float64:
				default:
					s := fmt.Sprintf("%v", value)
					f, err := strconv.ParseFloat(s, 64)
					if err != nil {
						logs.Error(err)
					} else {
						(*data)[key] = f
					}
				}
			case TypeBool:
				switch value.(type) {
				case bool:
				default:
					s := fmt.Sprintf("%v", value)
					if s == "1" || s == "true" {
						(*data)[key] = true
					} else {
						(*data)[key] = false
					}
				}
			case TypeObject:
				switch value.(type) {
				case map[string]interface{}:
				default:
					return fmt.Errorf("the property [%s] is not map[string]interface{} [%v]", key, value)
				}
			case TypeDate:
				(*data)[key] = value
			default:
			}
		}
	}
	return nil
}
