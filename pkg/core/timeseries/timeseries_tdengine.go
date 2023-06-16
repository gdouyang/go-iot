package timeseries

import (
	"bytes"
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/core/eventbus"
	"go-iot/pkg/core/tsl"
	"go-iot/pkg/core/util"
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
	err := t.dml("create database if not exists goiot;")
	if err != nil {
		return err
	}
	if len(model.Properties) > 0 {
		// 属性
		sb := strings.Builder{}
		sb.WriteString("CREATE STABLE IF NOT EXISTS ")
		sb.WriteString(t.getStableName(product, core.TIME_TYPE_PROP))
		sb.WriteString(" (")
		sb.WriteString(t.columnNameRewrite("createTime", "TIMESTAMP"))
		for _, p := range model.Properties {
			sb.WriteString(", ")
			t.createSqlColumn(&sb, p.Id, p)
		}
		sb.WriteString(" ) tags (")
		sb.WriteString(t.columnNameRewrite("deviceId", "nchar(64)"))
		sb.WriteString(");")
		err := t.dml(sb.String())
		if err != nil {
			return err
		}
	}
	{
		// 事件
		for _, e := range model.Events {
			sb := strings.Builder{}
			sb.WriteString("CREATE STABLE IF NOT EXISTS ")
			sb.WriteString(t.getEventStableName(product, core.TIME_TYPE_EVENT, e.Id))
			sb.WriteString(" (")
			sb.WriteString(t.columnNameRewrite("createTime", "TIMESTAMP, "))
			if e.IsObject() {
				object := e.ToValueTypeObject()
				for idx, p1 := range object.Properties {
					t.createSqlColumn(&sb, p1.Id, p1)
					if idx < len(object.Properties)-1 {
						sb.WriteString(", ")
					}
				}
			} else {
				t.createSqlColumn(&sb, e.Id, e)
			}
			sb.WriteString(" ) tags (")
			sb.WriteString(t.columnNameRewrite("deviceId", "nchar(64)"))
			sb.WriteString(");")
			err := t.dml(sb.String())
			if err != nil {
				return err
			}
		}
	}
	{
		// device logs
		sb := strings.Builder{}
		sb.WriteString("CREATE STABLE IF NOT EXISTS ")
		sb.WriteString(t.getStableName(product, core.TIME_TYPE_LOGS))
		sb.WriteString(" (")
		sb.WriteString(t.columnNameRewrite("createTime", "TIMESTAMP, "))
		sb.WriteString(t.columnNameRewrite("content", "nchar(1024), "))
		sb.WriteString(t.columnNameRewrite("type", "nchar(32)"))
		sb.WriteString(" ) tags (")
		sb.WriteString(t.columnNameRewrite("deviceId", "nchar(64)"))
		sb.WriteString(");")
		err := t.dml(sb.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TdengineTimeSeries) Del(product *core.Product) error {
	t.dml("DROP STABLE IF EXISTS " + t.getStableName(product, core.TIME_TYPE_PROP) + ";")
	for _, e := range product.TslData.Events {
		t.dml("DROP STABLE IF EXISTS " + t.getEventStableName(product, core.TIME_TYPE_EVENT, e.Id) + ";")
	}
	t.dml("DROP STABLE IF EXISTS " + t.getStableName(product, core.TIME_TYPE_LOGS) + ";")
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
	sb.WriteString(tableName)
	sb.WriteString(" where ")
	sb.WriteString(t.columnNameRewrite("deviceId"))
	sb.WriteString(" = ")
	sb.WriteString(t.whereValueRewrite(param.DeviceId))
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

	list := []map[string]any{}
	total, err := t.count("select count(*) from " + sb.String())
	if err != nil {
		return nil, err
	}
	if total > 0 {
		list, err = t.search("select * from " + sb.String())
		if err != nil {
			return nil, err
		}
	}

	var result map[string]any = map[string]any{
		"pageNum":    param.PageNum,
		"totalCount": total,
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
	sql := t.insertSql(sTableName, core.TIME_TYPE_PROP, columns, d1, time.Now().Format(timeformt))
	err := t.dml(sql)
	if err != nil {
		logs.Error("exec: %s", err)
	}
	event := eventbus.NewPropertiesMessage(fmt.Sprintf("%v", deviceId), product.GetId(), d1)
	eventbus.PublishProperties(&event)
	return nil
}

func (t *TdengineTimeSeries) SaveEvents(product *core.Product, eventId string, d1 map[string]any) error {
	eventMap := product.GetTsl().EventsMap()
	if eventMap == nil {
		return errors.New("not have tsl property, don't save timeseries data")
	}
	property, ok := eventMap[eventId]
	if !ok {
		return fmt.Errorf("eventId [%s] not found", eventId)
	}
	columns := []string{}
	if property.IsObject() {
		validProperty := property.PropertiesMap()
		for key := range d1 {
			if key == tsl.PropertyDeviceId {
				continue
			}
			if _, ok := validProperty[key]; ok {
				columns = append(columns, key)
			}
		}
	} else {
		for key := range d1 {
			if key == tsl.PropertyDeviceId {
				continue
			}
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
	sql := t.insertSql(sTableName, core.TIME_TYPE_EVENT, columns, d1, time.Now().Format(timeformt))
	err := t.dml(sql)
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
	columns := []string{"type", "content"}
	sTableName := t.getStableName(product, core.TIME_TYPE_LOGS)
	sql := t.insertSql(sTableName, core.TIME_TYPE_LOGS, columns, map[string]any{
		"type":     d1.Type,
		"deviceId": d1.DeviceId,
		"content":  d1.Content,
	}, d1.CreateTime)
	err := t.dml(sql)
	if err != nil {
		logs.Error("exec: %s", err)
	}
	return nil
}

// devicelogs-{productId}, properties-{productId}
func (t *TdengineTimeSeries) getStableName(product *core.Product, typ string) string {
	index := "goiot" + "." + typ + "_" + strings.ReplaceAll(product.GetId(), "-", "_")
	return index
}

// event-{productId}-{eventId}
func (t *TdengineTimeSeries) getEventStableName(product *core.Product, typ string, eventId string) string {
	index := "goiot" + "." + typ + "_" + strings.ReplaceAll(product.GetId(), "-", "_") + "_" + strings.ReplaceAll(eventId, "-", "_")
	return index
}

func (t *TdengineTimeSeries) getClient(sql string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:6041/rest/sql",
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
	return buf, nil
}

func (t *TdengineTimeSeries) dml(sql string) error {
	buf, err := t.getClient(sql)
	if err != nil {
		return err
	}

	if logs.GetBeeLogger().GetLevel() == logs.LevelDebug {
		// logs.Debug("<==", "rows:", gjson.GetBytes(buf, "rows"))
		logs.Debug("<==", "affected_rows:", gjson.GetBytes(buf, "data.0.0"))
	}
	code := gjson.GetBytes(buf, "code")
	if code.Raw != "0" {
		logs.Error(string(buf))
		return err
	}
	return nil
}

func (t *TdengineTimeSeries) count(sql string) (int64, error) {
	var total int64
	buf, err := t.getClient(sql)
	if err != nil {
		return total, err
	}
	// if logs.GetBeeLogger().GetLevel() == logs.LevelDebug {
	// 	logs.Debug("<==", "rows:", gjson.GetBytes(buf, "rows"))
	// }
	code := gjson.GetBytes(buf, "code")
	if code.Raw == "0" {
		total = gjson.GetBytes(buf, "data.0.0").Int()
		if logs.GetBeeLogger().GetLevel() == logs.LevelDebug {
			logs.Debug("<==", "columns:", gjson.GetBytes(buf, "column_meta.0.0"))
			logs.Debug("<==", "        ", total)
		}
	} else {
		logs.Error(string(buf))
		return 0, err
	}
	return total, nil
}

func (t *TdengineTimeSeries) search(sql string) ([]map[string]any, error) {
	buf, err := t.getClient(sql)
	if err != nil {
		return nil, err
	}

	// if logs.GetBeeLogger().GetLevel() == logs.LevelDebug {
	// 	logs.Debug("<==", "rows:", gjson.GetBytes(buf, "rows"))
	// }
	code := gjson.GetBytes(buf, "code")
	result := []map[string]any{}
	if code.Raw == "0" {
		data := gjson.GetBytes(buf, "data").Array()
		columns := []string{}
		values := [][]string{}
		for _, row := range data {
			item := map[string]any{}
			rowValue := []string{}
			for idx, val := range row.Array() {
				if !val.Exists() {
					continue
				}
				columnName := gjson.GetBytes(buf, fmt.Sprintf("column_meta.%v.0", idx))
				if logs.GetBeeLogger().GetLevel() == logs.LevelDebug {
					columns = append(columns, columnName.Raw)
					rowValue = append(rowValue, val.Raw)
				}
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
					value = val.String()
				}
				colName_ := columnName.String()
				if strings.Contains(colName_, "_0_") {
					arr := strings.Split(colName_, "_0_")
					objName := util.FirstLowCamelString(arr[0])
					if _, ok := item[objName]; !ok {
						item[objName] = map[string]any{}
					}
					propertyName := util.FirstLowCamelString(arr[1])
					item[objName].(map[string]any)[propertyName] = value
				} else {
					propertyName := util.FirstLowCamelString(colName_)
					item[propertyName] = value
				}
			}
			result = append(result, item)
			values = append(values, rowValue)
		}
		if logs.GetBeeLogger().GetLevel() == logs.LevelDebug {
			logs.Debug("<==", "columns:", strings.Join(columns, ","))
			for _, datas := range values {
				logs.Debug("<==", "        ", strings.Join(datas, ","))
			}
		}
	} else {
		logs.Error(string(buf))
		return nil, err
	}
	return result, nil
}

func (t *TdengineTimeSeries) createSqlColumn(sb *strings.Builder, columnName string, p tsl.TslProperty) {
	valType := strings.TrimSpace(p.Type)
	if valType == tsl.TypeObject {
		object := p.ValueType.(tsl.ValueTypeObject)
		for idx, p1 := range object.Properties {
			t.createSqlColumn(sb, p.Id+"."+p1.Id, p1)
			if idx < len(object.Properties)-1 {
				sb.WriteString(", ")
			}
		}
		return
	}
	sb.WriteString(t.columnNameRewrite(columnName))
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

func (t *TdengineTimeSeries) insertSql(sTableName string, type_ string, columns []string, data map[string]any, createTime string) string {
	sb := strings.Builder{}
	deviceId := data[tsl.PropertyDeviceId]
	sb.WriteString("INSERT INTO goiot.")
	sb.WriteString(type_)
	sb.WriteString(fmt.Sprintf("_%v ", deviceId))
	sb.WriteString("USING ")
	sb.WriteString(sTableName)
	sb.WriteString(" TAGS(")
	sb.WriteString(fmt.Sprintf("'%v'", deviceId))
	sb.WriteString(") ")
	sb.WriteString("( ")
	sb.WriteString(t.columnNameRewrite("createTime"))
	values := strings.Builder{}
	if len(columns) > 0 {
		sb.WriteString(",")
		values.WriteString(",")
	}
	for idx, col := range columns {
		sb.WriteString(t.columnNameRewrite(col))
		values.WriteString(t.whereValueRewrite(data[col]))
		if idx < len(columns)-1 {
			sb.WriteString(",")
			values.WriteString(",")
		}
	}
	sb.WriteString(") ")
	sb.WriteString("VALUES(")
	sb.WriteString(fmt.Sprintf("'%s'", createTime))
	sb.WriteString(values.String())
	sb.WriteString(");")
	return sb.String()
}

func (t *TdengineTimeSeries) whereValueRewrite(value any) string {
	switch value.(type) {
	case string:
		return "'" + strings.ReplaceAll(fmt.Sprintf("%v", value), "'", "\\'") + "'"
	default:
		return strings.ReplaceAll(fmt.Sprintf("%v", value), "'", "\\'")
	}
}

func (t *TdengineTimeSeries) columnNameRewrite(value string, type_ ...string) string {
	value = strings.ReplaceAll(value, ".", "_0_")
	if len(type_) > 0 {
		return util.SnakeString(value+"_") + " " + type_[0]
	}
	return util.SnakeString(value + "_")
}

func (t *TdengineTimeSeries) where(sb *strings.Builder, terms []core.SearchTerm) {
	for _, _term := range terms {
		if _term.Value == nil {
			continue
		}
		key := t.columnNameRewrite(_term.Key)
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
					sb.WriteString(t.whereValueRewrite(vv.Index(i).Interface()))
					if i < vv.Len()-1 {
						sb.WriteString(",")
					}
				}
			} else {
				s := fmt.Sprintf("%v", value)
				vals := strings.Split(s, ",")
				for idx, v := range vals {
					sb.WriteString(t.whereValueRewrite(v))
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
					sb.WriteString(t.whereValueRewrite(vv.Index(i).Interface()))
				}
			} else {
				s := fmt.Sprintf("%v", value)
				vals := strings.Split(s, ",")
				if len(vals) > 0 {
					sb.WriteString(" >= ")
					sb.WriteString(t.whereValueRewrite(vals[0]))
				}
				if len(vals) > 1 {
					sb.WriteString(" AND ")
					sb.WriteString(key)
					sb.WriteString(" <= ")
					sb.WriteString(t.whereValueRewrite(vals[1]))
				}
			}
		case core.LIKE:
			sb.WriteString(" LIKE ")
			sb.WriteString(t.whereValueRewrite(value))
		case core.GT:
			sb.WriteString(" > ")
			sb.WriteString(t.whereValueRewrite(value))
		case core.GTE:
			sb.WriteString(" >= ")
			sb.WriteString(t.whereValueRewrite(value))
		case core.LT:
			sb.WriteString(" < ")
			sb.WriteString(t.whereValueRewrite(value))
		case core.LTE:
			sb.WriteString(" <= ")
			sb.WriteString(t.whereValueRewrite(value))
		case core.NEQ:
			sb.WriteString(" != ")
			sb.WriteString(t.whereValueRewrite(value))
		default:
			sb.WriteString(" = ")
			sb.WriteString(t.whereValueRewrite(value))
		}
	}
}
