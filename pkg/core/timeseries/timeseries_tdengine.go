package timeseries

import (
	"bytes"
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/core/eventbus"
	"go-iot/pkg/core/tsl"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/tidwall/gjson"
)

// es时序保存
type TdengineTimeSeries struct {
}

func (t *TdengineTimeSeries) Id() string {
	return "tdengine"
}

func (t *TdengineTimeSeries) PublishModel(product *core.Product, model tsl.TslData) error {
	if len(model.Properties) > 0 {
		// 属性
		sb := strings.Builder{}
		sb.WriteString("CREATE STABLE IF NOT EXISTS ")
		sb.WriteString(t.getStableName(product, core.TIME_TYPE_PROP))
		sb.WriteString(" (createTime TIMESTAMP")
		for _, p := range model.Properties {
			sb.WriteString(", ")
			t.appendSqlColumn(&sb, p.Id, p)
		}
		sb.WriteString(" ) tags (deviceId nchar(64));")
		_, err := t.exec(sb.String())
		if err != nil {
			return err
		}
	}
	{
		// 事件
		for _, e := range model.Events {
			if len(e.Properties) > 0 {
				sb := strings.Builder{}
				sb.WriteString("CREATE STABLE IF NOT EXISTS ")
				sb.WriteString(t.getEventStableName(product, core.TIME_TYPE_EVENT, e.Id))
				sb.WriteString(" (createTime TIMESTAMP")
				for _, p := range e.Properties {
					sb.WriteString(", ")
					t.appendSqlColumn(&sb, p.Id, p)
				}
				sb.WriteString(") tags (deviceId nchar(64));")
				_, err := t.exec(sb.String())
				if err != nil {
					return err
				}
			}
		}
	}
	{
		// device logs
		sb := strings.Builder{}
		sb.WriteString("CREATE STABLE IF NOT EXISTS ")
		sb.WriteString(t.getStableName(product, core.TIME_TYPE_LOGS))
		sb.WriteString(" (createTime TIMESTAMP, content nchar(1024) ")
		sb.WriteString(") tags (deviceId nchar(64), type nchar(32));")
		_, err := t.exec(sb.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TdengineTimeSeries) Del(product *core.Product) error {
	t.exec("DROP STABLE IF EXISTS " + product.Id + ";")
	return nil
}

func (t *TdengineTimeSeries) QueryProperty(product *core.Product, param core.TimeDataSearchRequest) (map[string]any, error) {
	return t.query(t.getStableName(product, core.TIME_TYPE_PROP), param)
}

func (t *TdengineTimeSeries) QueryLogs(product *core.Product, param core.TimeDataSearchRequest) (map[string]any, error) {
	return t.query(t.getStableName(product, core.TIME_TYPE_LOGS), param)
}

func (t *TdengineTimeSeries) QueryEvent(product *core.Product, eventId string, param core.TimeDataSearchRequest) (map[string]interface{}, error) {
	return t.query(t.getEventStableName(product, core.TIME_TYPE_EVENT, eventId), param)
}

func (t *TdengineTimeSeries) query(tableName string, param core.TimeDataSearchRequest) (map[string]any, error) {
	if len(param.DeviceId) == 0 {
		return nil, errors.New("deviceId must be persent")
	}
	sb := strings.Builder{}
	sb.WriteString("select * from ")
	sb.WriteString(tableName)
	sb.WriteString(" where deviceId = ")
	sb.WriteString(param.DeviceId)
	t.where(&sb, param.Condition)
	if param.PageNum <= 0 {
		param.PageNum = 1
	}
	if param.PageSize <= 0 {
		param.PageSize = 10
	}
	sb.WriteString(" limit ")
	sb.WriteString(fmt.Sprintf("%v", param.PageOffset()))
	sb.WriteString(",")
	sb.WriteString(fmt.Sprintf("%v", param.PageSize))
	sb.WriteString(";")
	list, err := t.exec(sb.String())
	if err != nil {
		return nil, err
	}

	var result map[string]any = map[string]any{
		"pageNum":    param.PageNum,
		"totalCount": 0,
		"list":       list,
	}
	return result, nil
}

func (t *TdengineTimeSeries) SaveProperties(product *core.Product, d1 map[string]any) error {
	validProperty := product.GetTsl().PropertiesMap()
	if validProperty == nil {
		return errors.New("not have tsl property, don't save timeseries data")
	}
	columns := []string{}
	for key := range d1 {
		if key == tsl.PropertyDeviceId {
			continue
		}
		if _, ok := validProperty[key]; ok {
			columns = append(columns, key)
		}
	}
	if len(columns) == 0 {
		return errors.New("data is empty, don't save timeseries data")
	}
	deviceId := d1[tsl.PropertyDeviceId]
	if deviceId == nil {
		return errors.New("not have deviceId, don't save timeseries data")
	}
	sTableName := t.getStableName(product, core.TIME_TYPE_PROP)
	// INSERT INTO d1001 USING meters TAGS('Beijing.Chaoyang', 2) VALUES('a');
	sql := t.insertSql(sTableName, columns, d1, time.Now().Format(timeformt))
	_, err := t.exec(sql)
	if err != nil {
		logs.Error("exec: %s", err)
	}
	event := eventbus.NewPropertiesMessage(fmt.Sprintf("%v", deviceId), product.GetId(), d1)
	eventbus.PublishProperties(&event)
	return nil
}

func (t *TdengineTimeSeries) SaveEvents(product *core.Product, eventId string, d1 map[string]any) error {
	validProperty := product.GetTsl().EventsMap()
	if validProperty == nil {
		return errors.New("not have tsl property, don't save timeseries data")
	}
	event, ok := validProperty[eventId]
	if !ok {
		return fmt.Errorf("eventId [%s] not found", eventId)
	}
	columns := []string{}
	emap := event.PropertiesMap()
	for key := range d1 {
		if key == tsl.PropertyDeviceId {
			continue
		}
		if _, ok := emap[key]; ok {
			columns = append(columns, key)
		}
	}
	if len(columns) == 0 {
		return errors.New("data is empty, don't save event timeseries data")
	}
	deviceId := d1[tsl.PropertyDeviceId]
	if deviceId == nil {
		return errors.New("not have deviceId, don't save event timeseries data")
	}
	sTableName := t.getEventStableName(product, core.TIME_TYPE_EVENT, eventId)
	// // INSERT INTO d1001 USING meters TAGS('Beijing.Chaoyang', 2) VALUES('a');
	sql := t.insertSql(sTableName, columns, d1, time.Now().Format(timeformt))
	_, err := t.exec(sql)
	if err != nil {
		logs.Error("exec: %s", err)
	}
	evt := eventbus.NewEventMessage(fmt.Sprintf("%v", deviceId), product.GetId(), d1)
	eventbus.PublishEvent(&evt)
	return nil
}

func (t *TdengineTimeSeries) SaveLogs(product *core.Product, d1 core.LogData) error {
	if len(d1.DeviceId) == 0 {
		return errors.New("deviceId must be present, don't save logs timeseries data")
	}
	if len(d1.CreateTime) == 0 {
		d1.CreateTime = time.Now().Format(timeformt)
	}
	// Build the request body.
	columns := []string{}
	sTableName := t.getStableName(product, core.TIME_TYPE_LOGS)
	sql := t.insertSql(sTableName, columns, map[string]any{
		"type":     d1.Type,
		"deviceId": d1.DeviceId,
		"content":  d1.Content,
	}, d1.CreateTime)
	_, err := t.exec(sql)
	if err != nil {
		logs.Error("exec: %s", err)
	}
	return nil
}

// devicelogs-{productId}, properties-{productId}
func (t *TdengineTimeSeries) getStableName(product *core.Product, typ string) string {
	index := t.replace(typ + "_" + product.GetId())
	return index
}

// event-{productId}-{eventId}
func (t *TdengineTimeSeries) getEventStableName(product *core.Product, typ string, eventId string) string {
	index := t.replace(typ + "_" + product.GetId() + "_" + eventId)
	return index
}

func (t *TdengineTimeSeries) replace(str string) string {
	return strings.ReplaceAll(strings.ReplaceAll(str, "-", "_"), ".", "_")
}

func (t *TdengineTimeSeries) exec(sql string) ([]map[string]any, error) {
	req, err := http.NewRequest(http.MethodPost, "http://192.168.31.197:6041/rest/sql/test",
		bytes.NewBuffer([]byte(sql)))
	if err != nil {
		return nil, err
	}
	if logs.GetBeeLogger().GetLevel() == logs.LevelDebug {
		logs.Debug("==>", "  SQL:", sql)
	}
	req.Header.Add("Authorization", "Basic cm9vdDp0YW9zZGF0YQ==")
	req.Close = true

	client := &http.Client{Timeout: time.Duration(time.Second * 3)}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if logs.GetBeeLogger().GetLevel() == logs.LevelDebug {
		logs.Debug("<==", "Total:", gjson.GetBytes(buf, "rows"))
	}
	code := gjson.GetBytes(buf, "code")
	result := []map[string]any{}
	if code.Raw == "0" {
		data := gjson.GetBytes(buf, "data").Array()
		for _, row := range data {
			for idx, val := range row.Array() {
				columnName := gjson.GetBytes(buf, fmt.Sprintf("column_meta.%v.0", idx))
				columnType := gjson.GetBytes(buf, fmt.Sprintf("column_meta.%v.1", idx))
				var value any
				switch columnType.Raw {
				case "TINYINT":
				case "SMALLINT":
				case "INT":
				case "BIGINT":
					value = val.Int()
				case "TINYINT UNSIGNED":
				case "SMALLINT UNSIGNED":
				case "INT UNSIGNED":
				case "BIGINT UNSIGNED":
					value = val.Uint()
				case "BOOL":
					value = val.Bool()
				case "DOUBLE":
				case "FLOAT":
					value = val.Float()
				default:
					value = val.Raw
				}
				item := map[string]any{columnName.Raw: value}
				result = append(result, item)
			}
		}
	} else {
		logs.Error(string(buf))
		return nil, err
	}
	return result, nil
}

func (t *TdengineTimeSeries) appendSqlColumn(sb *strings.Builder, columnName string, p tsl.TslProperty) {
	valType := strings.TrimSpace(p.Type)
	if valType == tsl.TypeObject {
		object := p.ValueType.(tsl.ValueTypeObject)
		for idx, p1 := range object.Properties {
			t.appendSqlColumn(sb, p.Id+"_"+p1.Id, p1)
			if idx < len(object.Properties)-1 {
				sb.WriteString(", ")
			}
		}
		return
	}
	sb.WriteString(columnName)
	switch valType {
	case tsl.TypeInt:
		sb.WriteString(" INT")
	case tsl.TypeLong:
		sb.WriteString(" BIGINT")
	case tsl.TypeFloat:
		sb.WriteString(" FLOAT")
	case tsl.TypeDouble:
		sb.WriteString(" DOUBLE")
	case tsl.TypeBool:
		sb.WriteString(" BOOL")
	case tsl.TypeEnum:
		sb.WriteString(" NCHAR(32)")
	case tsl.TypeString:
		sb.WriteString(" NCHAR(32)")
	case tsl.TypePassword:
		sb.WriteString(" NCHAR(32)")
	case tsl.TypeDate:
		sb.WriteString(" TIMESTAMP")
	// case tsl.TypeArray:
	// array := p.ValueType.(tsl.ValueTypeArray)
	// return t.appendSqlColumn(array.ElementType)
	default:
		if len(p.Id) > 0 {
			sb.WriteString(" NCHAR(32)")
		}
	}
}

func (t *TdengineTimeSeries) insertSql(sTableName string, columns []string, data map[string]any, createTime string) string {
	sb := strings.Builder{}
	deviceId := data[tsl.PropertyDeviceId]
	sb.WriteString("INSERT ")
	sb.WriteString(fmt.Sprintf("%v ", deviceId))
	sb.WriteString(sTableName)
	sb.WriteString(" TAGS(")
	sb.WriteString(fmt.Sprintf("'%s', ", createTime))
	sb.WriteString(fmt.Sprintf("'%v'", deviceId))
	sb.WriteString(") ")
	sb.WriteString("( ")
	sb.WriteString(strings.Join(columns, ","))
	sb.WriteString(") ")
	sb.WriteString("VALUES(")
	for idx, col := range columns {
		value := data[col]
		sb.WriteString(t.strescap(value))
		if idx < len(columns)-1 {
			sb.WriteString(",")
		}
	}
	sb.WriteString(") ")
	return sb.String()
}

func (t *TdengineTimeSeries) strescap(value any) string {
	switch value.(type) {
	case string:
		return "'" + t.escap(fmt.Sprintf("%v", value)) + "'"
	default:
		return t.escap(fmt.Sprintf("%v", value))
	}
}

func (t *TdengineTimeSeries) escap(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

func (t *TdengineTimeSeries) where(sb *strings.Builder, terms []core.SearchTerm) {
	for _, _term := range terms {
		if _term.Value == nil {
			continue
		}
		key := t.replace(_term.Key)
		value := _term.Value
		sb.WriteString(" AND ")
		sb.WriteString(key)
		switch _term.Oper {
		case core.IN:
			sb.WriteString(" IN ( ")
			kind := reflect.TypeOf(value).Kind()
			if kind == reflect.Array || kind == reflect.Slice {
				vv := reflect.ValueOf(value)
				for i := 0; i < vv.Len(); i++ {
					sb.WriteString(t.strescap(vv.Index(i).Interface()))
					if i < vv.Len()-1 {
						sb.WriteString(",")
					}
				}
			} else {
				s := fmt.Sprintf("%v", value)
				vals := strings.Split(s, ",")
				for idx, v := range vals {
					sb.WriteString(t.strescap(v))
					if idx < len(vals)-1 {
						sb.WriteString(",")
					}
				}
			}
			sb.WriteString(")")
		case core.BTW:
			kind := reflect.TypeOf(value).Kind()
			if kind == reflect.Array || kind == reflect.Slice {
				vv := reflect.ValueOf(value)
				for i := 0; i < vv.Len(); i++ {
					if i == 0 {
						sb.WriteString(" >= ")
					} else if i == 1 {
						sb.WriteString(" AND ")
						sb.WriteString(key)
						sb.WriteString(" <= ")
					} else {
						break
					}
					sb.WriteString(t.strescap(vv.Index(i).Interface()))
				}
			} else {
				s := fmt.Sprintf("%v", value)
				vals := strings.Split(s, ",")
				if len(vals) > 0 {
					sb.WriteString(" >= ")
					sb.WriteString(t.escap(vals[0]))
				}
				if len(vals) > 1 {
					sb.WriteString(" AND ")
					sb.WriteString(key)
					sb.WriteString(" <= ")
					sb.WriteString(t.escap(vals[1]))
				}
			}
		case core.LIKE:
			sb.WriteString(" LIKE ")
			sb.WriteString(t.strescap(value))
		case core.GT:
			sb.WriteString(" > ")
			sb.WriteString(t.strescap(value))
		case core.GTE:
			sb.WriteString(" >= ")
			sb.WriteString(t.strescap(value))
		case core.LT:
			sb.WriteString(" < ")
			sb.WriteString(t.strescap(value))
		case core.LTE:
			sb.WriteString(" <= ")
			sb.WriteString(t.strescap(value))
		case core.NEQ:
			sb.WriteString(" != ")
			sb.WriteString(t.strescap(value))
		default:
			sb.WriteString(" = ")
			sb.WriteString(t.strescap(value))
		}
	}
}
