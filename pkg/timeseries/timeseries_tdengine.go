package timeseries

import (
	"bytes"
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/tsl"
	"go-iot/pkg/util"
	"io"
	"net/http"
	"reflect"
	"strings"

	logs "go-iot/pkg/logger"

	"github.com/tidwall/gjson"
)

func init() {
	core.RegisterTimeSeries(&TdengineTimeSeries{})
}

// Tdengine时序数据保存
type TdengineTimeSeries struct {
}

func (t *TdengineTimeSeries) Id() string {
	return core.TIME_SERISE_TDENGINE
}

// PublishModel 实现见 tdengine_schema.go（支持物模型增删列同步）

func (t *TdengineTimeSeries) Del(product *core.Product) error {
	_ = t.dml("DROP STABLE IF EXISTS " + t.getStableName(product, core.TIME_TYPE_PROP) + ";")
	// 按前缀清理全部 event 表（含物模型已删除的残留事件表）
	prefix := "event_" + tdNamePart(product.GetId()) + "_"
	if names, err := t.listStablesWithPrefix(prefix); err == nil {
		for _, name := range names {
			_ = t.dml("DROP STABLE IF EXISTS " + t.dbPrefix() + name + ";")
		}
	}
	if product.TslData != nil {
		for _, e := range product.TslData.Events {
			_ = t.dml("DROP STABLE IF EXISTS " + t.getEventStableName(product, core.TIME_TYPE_EVENT, e.GetId()) + ";")
		}
	}
	_ = t.dml("DROP STABLE IF EXISTS " + t.getStableName(product, core.TIME_TYPE_LOGS) + ";")
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
	if param.PageNum <= 0 {
		param.PageNum = 1
	}
	if param.PageSize <= 0 {
		param.PageSize = 10
	}

	// where 子句（不含 limit），count 与 search 共用
	where := strings.Builder{}
	where.WriteString(tableName)
	where.WriteString(" where ")
	where.WriteString(t.columnNameRewrite("deviceId"))
	where.WriteString(" = ")
	where.WriteString(t.whereValueRewrite(param.DeviceId))
	t.where(&where, param.Condition)
	whereSQL := where.String()

	list := []map[string]any{}
	total, err := t.count("select count(*) from " + whereSQL + ";")
	if err != nil {
		return nil, err
	}
	if total > 0 {
		searchSQL := fmt.Sprintf("select * from %s limit %v,%v;", whereSQL, param.PageOffset(), param.PageSize)
		list, err = t.search(searchSQL)
		if err != nil {
			return nil, err
		}
	}

	return pageQueryResult(total, param.PageNum, param.PageSize, list), nil
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
	createTime := nextTdCreateTime()
	// 入批量队列（异步合并提交），校验已在上方完成
	frag := t.insertFragment(sTableName, core.TIME_TYPE_PROP, columns, d1, createTime)
	defaultTdBatch.commit(frag)
	// 发送事件总线（与 ES 异步落盘语义一致：入队成功即发布）
	d1["createTime"] = createTime
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
	if obj, ok := property.IsObject(); ok {
		validProperty := obj.PropertiesMap()
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
	createTime := nextTdCreateTime()
	frag := t.insertFragment(sTableName, core.TIME_TYPE_EVENT, columns, d1, createTime)
	defaultTdBatch.commit(frag)
	d1["createTime"] = createTime
	// 发送事件总线
	evt := eventbus.NewEventMessage(fmt.Sprintf("%v", deviceId), product.GetId(), eventId, d1)
	eventbus.PublishEvent(&evt)
	return nil
}

func (t *TdengineTimeSeries) SaveLogs(product *core.Product, d1 core.LogData) error {
	if len(d1.DeviceId) == 0 {
		return errors.New("deviceId must be present, don't save logs timeseries data")
	}
	if len(d1.CreateTime) == 0 {
		d1.CreateTime = nextTdCreateTime()
	}
	columns := []string{"type", "content"}
	sTableName := t.getStableName(product, core.TIME_TYPE_LOGS)
	frag := t.insertFragment(sTableName, core.TIME_TYPE_LOGS, columns, map[string]any{
		"type":     d1.Type,
		"deviceId": d1.DeviceId,
		"content":  d1.Content,
	}, d1.CreateTime)
	defaultTdBatch.commit(frag)
	return nil
}

// {database}.properties_{productId} / {database}.devicelogs_{productId}
// 产品 ID 经 tdNamePart：小写，'-'→'_'
func (t *TdengineTimeSeries) getStableName(product *core.Product, typ string) string {
	return tdDatabase() + "." + typ + "_" + tdNamePart(product.GetId())
}

// {database}.event_{productId}_{eventId}
func (t *TdengineTimeSeries) getEventStableName(product *core.Product, typ string, eventId string) string {
	return tdDatabase() + "." + typ + "_" + tdNamePart(product.GetId()) + "_" + tdNamePart(eventId)
}

func (t *TdengineTimeSeries) dbPrefix() string {
	return tdDatabase() + "."
}

func (t *TdengineTimeSeries) getClient(sql string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, tdRestSQLURL(), bytes.NewBuffer([]byte(sql)))
	if err != nil {
		return nil, err
	}
	logs.Debugf("==>  SQL:%s", sql)
	req.Header.Set("Authorization", tdAuthHeader())
	req.Close = true

	client := &http.Client{Timeout: tdHTTPTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
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
	code := gjson.GetBytes(buf, "code")
	if logs.IsDebug() && code.Raw == "0" {
		sql := strings.TrimSpace(sql)
		logs.Debugf("<== %s %s", strings.Split(sql, " ")[0], "OK")
	}
	if code.Raw != "0" {
		logs.Errorf(string(buf))
		return errors.New(string(buf))
	}
	return nil
}

func (t *TdengineTimeSeries) insert(sql string) error {
	buf, err := t.getClient(sql)
	if err != nil {
		return err
	}

	code := gjson.GetBytes(buf, "code")
	if logs.IsDebug() && code.Raw == "0" {
		logs.Debugf("<== Insert OK, affected_rows: %v", gjson.GetBytes(buf, "data.0.0"))
	}
	if code.Raw != "0" {
		logs.Errorf(string(buf))
		return errors.New(string(buf))
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
		if logs.IsDebug() {
			logs.Debugf("<== columns:%v", gjson.GetBytes(buf, "column_meta.0.0"))
			logs.Debugf("<==         %v", total)
		}
	} else {
		logs.Errorf(string(buf))
		return 0, errors.New(string(buf))
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
				columnName := gjson.GetBytes(buf, fmt.Sprintf("column_meta.%v.0", idx))
				if logs.IsDebug() {
					columns = append(columns, columnName.Raw)
					rowValue = append(rowValue, val.Raw)
				}
				if val.Raw == "null" {
					continue
				}
				columnType := gjson.GetBytes(buf, fmt.Sprintf("column_meta.%v.1", idx))
				var value any
				switch columnType.String() {
				case "TINYINT":
					value = val.Int()
				case "SMALLINT":
					value = val.Int()
				case "INT":
					value = val.Int()
				case "BIGINT":
					value = val.Int()
				case "TINYINT UNSIGNED":
					value = val.Uint()
				case "SMALLINT UNSIGNED":
					value = val.Uint()
				case "INT UNSIGNED":
					value = val.Uint()
				case "BIGINT UNSIGNED":
					value = val.Uint()
				case "BOOL":
					value = val.Bool()
				case "DOUBLE":
					value = val.Float()
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
		if logs.IsDebug() {
			logs.Debugf("<== columns:%s", strings.Join(columns, ","))
			for _, datas := range values {
				logs.Debugf("<==         %s", strings.Join(datas, ","))
			}
		}
	} else {
		logs.Errorf(string(buf))
		return nil, errors.New(string(buf))
	}
	return result, nil
}

func (t *TdengineTimeSeries) createSqlColumn(sb *strings.Builder, columnName string, p tsl.Property) {
	valType := strings.TrimSpace(p.GetType())
	if valType == tsl.TypeObject {
		object := p.(*tsl.PropertyObject)
		for idx, p1 := range object.Properties {
			t.createSqlColumn(sb, p.GetId()+"."+p1.GetId(), p1)
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
		if len(p.GetId()) > 0 {
			sb.WriteString(" NCHAR(32)")
		}
	}
}

// insertFragment 生成可拼入批量 INSERT 的片段（不含 INSERT INTO / 分号）。
// 例：goiot.properties_d1 USING goiot.properties_p1 TAGS('d1') (create_time_,light_) VALUES('...',1)
func (t *TdengineTimeSeries) insertFragment(sTableName string, type_ string, columns []string, data map[string]any, createTime string) string {
	sb := strings.Builder{}
	deviceId := data[tsl.PropertyDeviceId]
	// 子表名：{type}_{deviceId}，事件额外带 stable 后缀避免多事件共用一张子表冲突
	child := fmt.Sprintf("%s_%v", type_, deviceId)
	if type_ == core.TIME_TYPE_EVENT {
		stable := sTableName
		if i := strings.LastIndex(sTableName, "."); i >= 0 {
			stable = sTableName[i+1:]
		}
		child = fmt.Sprintf("%s_%v", stable, deviceId)
	}
	sb.WriteString(t.dbPrefix())
	sb.WriteString(child)
	sb.WriteString(" USING ")
	sb.WriteString(sTableName)
	sb.WriteString(" TAGS(")
	sb.WriteString(fmt.Sprintf("'%v'", deviceId))
	sb.WriteString(") ")
	sb.WriteString("(")
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
	sb.WriteString(") VALUES(")
	sb.WriteString(fmt.Sprintf("'%s'", createTime))
	sb.WriteString(values.String())
	sb.WriteString(")")
	return sb.String()
}

// insertSql 单条完整 SQL（兼容旧测试 / 调试）。
func (t *TdengineTimeSeries) insertSql(sTableName string, type_ string, columns []string, data map[string]any, createTime string) string {
	return "INSERT INTO " + t.insertFragment(sTableName, type_, columns, data, createTime) + ";"
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
		case core.NOTNULL:
			sb.WriteString(" is not null ")
		default:
			sb.WriteString(" = ")
			sb.WriteString(t.whereValueRewrite(value))
		}
	}
}
