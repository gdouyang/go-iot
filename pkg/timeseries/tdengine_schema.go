package timeseries

import (
	"fmt"
	"strings"

	"go-iot/pkg/core"
	"go-iot/pkg/tsl"

	logs "go-iot/pkg/logger"

	"github.com/tidwall/gjson"
)

// tdColDef 期望的列定义（非 TAG）
type tdColDef struct {
	Name string // 物理列名 create_time_ / light_
	Type string // 规范化类型 INT / DOUBLE / NCHAR / ...
	DDL  string // 完整类型片段，如 "INT" / "NCHAR(32)"
}

// PublishModel：建库 + 同步 properties/events/logs 表结构（支持新增列、删除列、新事件表、清理废弃事件表）
func (t *TdengineTimeSeries) PublishModel(product *core.Product, model tsl.TslData) error {
	if err := t.dml(fmt.Sprintf("create database if not exists %s;", tdDatabase())); err != nil {
		return err
	}
	if err := t.ensurePropertiesStable(product, model); err != nil {
		return err
	}
	if err := t.ensureEventStables(product, model); err != nil {
		return err
	}
	if err := t.ensureLogsStable(product); err != nil {
		return err
	}
	if err := t.dropObsoleteEventStables(product, model); err != nil {
		// 清理失败不阻断发布，记日志
		logs.Warnf("tdengine drop obsolete event stables: %v", err)
	}
	return nil
}

func (t *TdengineTimeSeries) ensurePropertiesStable(product *core.Product, model tsl.TslData) error {
	stable := t.getStableName(product, core.TIME_TYPE_PROP)
	// 即使 properties 为空也建仅含 createTime 的表，便于后续 ADD COLUMN
	cols := []tdColDef{{Name: t.columnNameRewrite("createTime"), Type: "TIMESTAMP", DDL: "TIMESTAMP"}}
	for _, p := range model.Properties {
		cols = append(cols, t.expandPropertyColumns(p.GetId(), p)...)
	}
	return t.syncStable(stable, cols, true)
}

func (t *TdengineTimeSeries) ensureEventStables(product *core.Product, model tsl.TslData) error {
	for _, e := range model.Events {
		stable := t.getEventStableName(product, core.TIME_TYPE_EVENT, e.GetId())
		cols := []tdColDef{{Name: t.columnNameRewrite("createTime"), Type: "TIMESTAMP", DDL: "TIMESTAMP"}}
		if object, ok := e.IsObject(); ok {
			for _, p1 := range object.Properties {
				cols = append(cols, t.expandPropertyColumns(p1.GetId(), p1)...)
			}
		} else {
			cols = append(cols, t.expandPropertyColumns(e.GetId(), e)...)
		}
		if err := t.syncStable(stable, cols, true); err != nil {
			return fmt.Errorf("event %s: %w", e.GetId(), err)
		}
	}
	return nil
}

func (t *TdengineTimeSeries) ensureLogsStable(product *core.Product) error {
	stable := t.getStableName(product, core.TIME_TYPE_LOGS)
	cols := []tdColDef{
		{Name: t.columnNameRewrite("createTime"), Type: "TIMESTAMP", DDL: "TIMESTAMP"},
		{Name: t.columnNameRewrite("content"), Type: "NCHAR", DDL: "NCHAR(1024)"},
		{Name: t.columnNameRewrite("type"), Type: "NCHAR", DDL: "NCHAR(32)"},
		{Name: t.columnNameRewrite("traceId"), Type: "NCHAR", DDL: "NCHAR(64)"},
	}
	// logs 固定 schema：只 ADD 缺失列，不 DROP（避免误删）
	return t.syncStable(stable, cols, false)
}

// syncStable 表不存在则 CREATE；存在则 ADD 缺失列，可选 DROP 多余列。
func (t *TdengineTimeSeries) syncStable(stable string, want []tdColDef, dropExtra bool) error {
	exist, err := t.stableExists(stable)
	if err != nil {
		return err
	}
	if !exist {
		return t.createStable(stable, want)
	}
	have, err := t.describeColumns(stable)
	if err != nil {
		return err
	}
	wantMap := map[string]tdColDef{}
	for _, c := range want {
		wantMap[c.Name] = c
	}
	// ADD
	for _, c := range want {
		ht, ok := have[c.Name]
		if !ok {
			sql := fmt.Sprintf("ALTER STABLE %s ADD COLUMN %s %s;", stable, c.Name, c.DDL)
			logs.Infof("tdengine schema ADD COLUMN: %s", sql)
			if err := t.dml(sql); err != nil {
				return fmt.Errorf("ADD COLUMN %s: %w", c.Name, err)
			}
			continue
		}
		if !tdTypesCompatible(ht, c.Type) {
			return fmt.Errorf("column %s type mismatch on %s: table=%s model=%s (change type not auto-migrated)",
				c.Name, stable, ht, c.Type)
		}
	}
	// DROP 多余普通列（保留 TAG / create_time_）
	if dropExtra {
		for name, ht := range have {
			if name == t.columnNameRewrite("createTime") {
				continue
			}
			// TAG 列在 describe 里 note=TAG，我们 describe 时标记 isTag
			if strings.HasSuffix(ht, "|TAG") {
				continue
			}
			if _, ok := wantMap[name]; ok {
				continue
			}
			sql := fmt.Sprintf("ALTER STABLE %s DROP COLUMN %s;", stable, name)
			logs.Infof("tdengine schema DROP COLUMN: %s", sql)
			if err := t.dml(sql); err != nil {
				return fmt.Errorf("DROP COLUMN %s: %w", name, err)
			}
		}
	}
	return nil
}

func (t *TdengineTimeSeries) createStable(stable string, cols []tdColDef) error {
	sb := strings.Builder{}
	sb.WriteString("CREATE STABLE IF NOT EXISTS ")
	sb.WriteString(stable)
	sb.WriteString(" (")
	for i, c := range cols {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(c.Name)
		sb.WriteString(" ")
		sb.WriteString(c.DDL)
	}
	sb.WriteString(" ) tags (")
	sb.WriteString(t.columnNameRewrite("deviceId", "nchar(64)"))
	sb.WriteString(");")
	logs.Infof("tdengine schema CREATE STABLE: %s", stable)
	return t.dml(sb.String())
}

func (t *TdengineTimeSeries) stableExists(stable string) (bool, error) {
	// stable: goiot.properties_xxx
	name := stable
	if i := strings.LastIndex(stable, "."); i >= 0 {
		name = stable[i+1:]
	}
	sql := fmt.Sprintf("SELECT stable_name FROM information_schema.ins_stables WHERE db_name='%s' AND stable_name='%s';",
		tdDatabase(), name)
	buf, err := t.getClient(sql)
	if err != nil {
		return false, err
	}
	code := gjson.GetBytes(buf, "code")
	if code.Raw != "0" {
		// 库/表元数据异常时退回 DESCRIBE
		return t.stableExistsByDescribe(stable)
	}
	rows := gjson.GetBytes(buf, "rows").Int()
	return rows > 0, nil
}

func (t *TdengineTimeSeries) stableExistsByDescribe(stable string) (bool, error) {
	buf, err := t.getClient("DESCRIBE " + stable)
	if err != nil {
		return false, err
	}
	code := gjson.GetBytes(buf, "code")
	if code.Raw == "0" {
		return true, nil
	}
	// 表不存在
	desc := gjson.GetBytes(buf, "desc").String()
	if strings.Contains(strings.ToLower(desc), "not exist") ||
		strings.Contains(strings.ToLower(string(buf)), "not exist") {
		return false, nil
	}
	return false, fmt.Errorf("describe %s: %s", stable, string(buf))
}

// describeColumns 返回 colName -> 规范化类型，TAG 列为 "TYPE|TAG"
func (t *TdengineTimeSeries) describeColumns(stable string) (map[string]string, error) {
	buf, err := t.getClient("DESCRIBE " + stable)
	if err != nil {
		return nil, err
	}
	if gjson.GetBytes(buf, "code").Raw != "0" {
		return nil, fmt.Errorf("describe %s: %s", stable, string(buf))
	}
	out := map[string]string{}
	data := gjson.GetBytes(buf, "data").Array()
	for _, row := range data {
		arr := row.Array()
		if len(arr) < 2 {
			continue
		}
		name := arr[0].String()
		typ := normalizeTdType(arr[1].String())
		note := ""
		if len(arr) >= 4 {
			note = arr[3].String()
		}
		if strings.EqualFold(note, "TAG") {
			typ = typ + "|TAG"
		}
		out[name] = typ
	}
	return out, nil
}

func (t *TdengineTimeSeries) dropObsoleteEventStables(product *core.Product, model tsl.TslData) error {
	// 期望保留的 event stable 短名
	keep := map[string]struct{}{}
	for _, e := range model.Events {
		full := t.getEventStableName(product, core.TIME_TYPE_EVENT, e.GetId())
		keep[stableShortName(full)] = struct{}{}
	}
	prefix := "event_" + tdNamePart(product.GetId()) + "_"
	names, err := t.listStablesWithPrefix(prefix)
	if err != nil {
		return err
	}
	for _, name := range names {
		if _, ok := keep[name]; ok {
			continue
		}
		sql := "DROP STABLE IF EXISTS " + tdDatabase() + "." + name + ";"
		logs.Infof("tdengine schema DROP obsolete event stable: %s", name)
		if err := t.dml(sql); err != nil {
			return err
		}
	}
	return nil
}

func (t *TdengineTimeSeries) listStablesWithPrefix(prefix string) ([]string, error) {
	// stable_name 精确前缀匹配
	sql := fmt.Sprintf("SELECT stable_name FROM information_schema.ins_stables WHERE db_name='%s' AND stable_name LIKE '%s%%';",
		tdDatabase(), strings.ReplaceAll(prefix, "'", "\\'"))
	buf, err := t.getClient(sql)
	if err != nil {
		return nil, err
	}
	if gjson.GetBytes(buf, "code").Raw != "0" {
		return nil, fmt.Errorf("list stables: %s", string(buf))
	}
	var out []string
	for _, row := range gjson.GetBytes(buf, "data").Array() {
		arr := row.Array()
		if len(arr) > 0 {
			out = append(out, arr[0].String())
		}
	}
	return out, nil
}

func stableShortName(full string) string {
	if i := strings.LastIndex(full, "."); i >= 0 {
		return full[i+1:]
	}
	return full
}

// expandPropertyColumns 展开属性（含 object 子字段）为物理列。
func (t *TdengineTimeSeries) expandPropertyColumns(columnName string, p tsl.Property) []tdColDef {
	valType := strings.TrimSpace(p.GetType())
	if valType == tsl.TypeObject {
		object, ok := p.IsObject()
		if !ok || object == nil {
			return nil
		}
		var cols []tdColDef
		for _, p1 := range object.Properties {
			cols = append(cols, t.expandPropertyColumns(columnName+"."+p1.GetId(), p1)...)
		}
		return cols
	}
	ddl := t.propertyTypeDDL(p)
	name := t.columnNameRewrite(columnName)
	return []tdColDef{{Name: name, Type: normalizeTdType(ddl), DDL: ddl}}
}

func (t *TdengineTimeSeries) propertyTypeDDL(p tsl.Property) string {
	switch strings.TrimSpace(p.GetType()) {
	case tsl.TypeInt:
		return "INT"
	case tsl.TypeLong:
		return "BIGINT"
	case tsl.TypeFloat:
		return "FLOAT"
	case tsl.TypeDouble:
		return "DOUBLE"
	case tsl.TypeBool:
		return "BOOL"
	case tsl.TypeEnum, tsl.TypeString, tsl.TypePassword:
		return "NCHAR(32)"
	case tsl.TypeDate:
		return "TIMESTAMP"
	default:
		if len(p.GetId()) > 0 {
			return "NCHAR(32)"
		}
		return "NCHAR(32)"
	}
}

func normalizeTdType(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	// NCHAR(32) / NCHAR -> NCHAR
	if strings.HasPrefix(s, "NCHAR") {
		return "NCHAR"
	}
	if strings.HasPrefix(s, "BINARY") {
		return "BINARY"
	}
	if strings.HasPrefix(s, "VARCHAR") {
		return "VARCHAR"
	}
	return s
}

func tdTypesCompatible(have, want string) bool {
	have = strings.TrimSuffix(have, "|TAG")
	return normalizeTdType(have) == normalizeTdType(want)
}
