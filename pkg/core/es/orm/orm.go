package orm

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/core/es"
	"reflect"
	"strings"
	"time"
)

var ErrNoRows = errors.New("no rows")
var ErrMissPK = errors.New("miss pk")
var ErrArgs = errors.New("err args")
var ErrNotModel = errors.New("model not reg")

func NewOrm() Orm {
	return &defaultOrm{}
}

type Orm interface {
	QueryTable(m interface{}) *QuerySeter
	Insert(md interface{}) (int64, error)
	InsertMulti(size int, md interface{}) (int64, error)
	Update(md interface{}, cols ...string) (int64, error)
	Delete(md interface{}, cols ...string) (int64, error)
	Read(md interface{}, cols ...string) error
}

type Params map[string]interface{}

type defaultOrm struct {
}

func (o *defaultOrm) QueryTable(md interface{}) *QuerySeter {
	mi, ok := defaultmodelCache.getByMd(md)
	if !ok {
		panic(fmt.Errorf("model not exist %d", md))
	}
	return &QuerySeter{
		indexName:  mi.indexName,
		model:      md,
		mi:         mi,
		filter:     map[string]interface{}{},
		pageSize:   10000,
		pageOffset: 0,
	}
}

type QuerySeter struct {
	indexName  string
	model      interface{}
	mi         *modelInfo
	filter     map[string]interface{}
	isQuery    bool
	total      int64
	pageSize   int
	pageOffset int
	orderBy    []orderBy
}

type orderBy struct {
	key  string
	sort string
}

func (q *QuerySeter) Filter(key string, value interface{}) *QuerySeter {
	key = strings.ReplaceAll(key, "__contains", "$LIKE")
	key = strings.ReplaceAll(key, "__in", "$IN")
	q.filter[FirstLower(key)] = value
	return q
}

func (q *QuerySeter) Count() (int64, error) {
	if q.isQuery {
		return q.total, nil
	}
	q.isQuery = true
	f := es.AppendFilter(q.filter)
	query := es.Query{From: 0, Size: 1, Filter: f}
	total, _, err := es.FilterSearch[map[string]interface{}](q.indexName, query)
	return total, err
}
func (q *QuerySeter) Limit(pageSize, pageOffset int) *QuerySeter {
	q.pageSize = pageSize
	q.pageOffset = pageOffset
	return q
}

func (q *QuerySeter) OrderBy(key string) *QuerySeter {
	sort := "asc"
	if strings.HasPrefix(key, "-") {
		key = FirstLower(strings.Replace(key, "-", "", 1))
		sort = "desc"
	} else if strings.HasPrefix(key, "+") {
		key = FirstLower(strings.Replace(key, "+", "", 1))
	}
	q.orderBy = append(q.orderBy, orderBy{key: key, sort: sort})
	return q
}

func (q *QuerySeter) Update(p Params) (int64, error) {
	f := es.AppendFilter(q.filter)
	sb := strings.Builder{}
	for key := range p {
		sb.WriteString("ctx._source['")
		sb.WriteString(FirstLower(key))
		sb.WriteString("']")
		sb.WriteString(" = ")
		sb.WriteString("params.")
		sb.WriteString(key)
	}
	script := map[string]interface{}{
		"source": sb.String(),
		"params": p,
	}
	return 1, es.UpdateDocByQuery(q.mi.indexName, f, script)
}

func (q *QuerySeter) All(result interface{}, cols ...string) (int64, error) {
	q.isQuery = true
	f := es.AppendFilter(q.filter)
	query := es.Query{From: q.pageOffset, Size: q.pageSize, Filter: f}
	for _, v := range q.orderBy {
		var o map[string]es.SortOrder = make(map[string]es.SortOrder)
		o[v.key] = es.SortOrder{Order: v.sort}
		query.Sort = append(query.Sort, o)
	}
	if len(cols) > 0 {
		for _, v := range cols {
			query.Includes = append(query.Includes, FirstLower(v))
		}

	}
	total, list, err := es.FilterSearch[map[string]interface{}](q.indexName, query)
	if err != nil {
		return 0, err
	}
	q.total = total
	length := len(list)
	if length > 0 {
		sb := strings.Builder{}
		sb.WriteString("[")
		for idx, v := range list {
			jsonstring, err := json.Marshal(v)
			if err != nil {
				return 0, err
			}
			sb.Write(jsonstring)
			if idx < length-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteString("]")
		err = json.Unmarshal([]byte(sb.String()), result)
	}
	return 0, err
}

func (o *defaultOrm) Insert(md interface{}) (int64, error) {
	mi, ok := defaultmodelCache.getByMd(md)
	if !ok {
		return 0, fmt.Errorf("model not exist %d", md)
	}
	docId := mi.getPKValue(md)
	if docId == "0" {
		id := time.Now().UnixMilli()
		docId = fmt.Sprintf("%v", id)
		mi.setFieldValue(md, mi.pkkey, id)
	}
	return 1, es.CreateDoc(mi.indexName, docId, md)
}

func (o *defaultOrm) InsertMulti(bulk int, mds interface{}) (int64, error) {
	var cnt int64

	sind := reflect.Indirect(reflect.ValueOf(mds))

	switch sind.Kind() {
	case reflect.Array, reflect.Slice:
		if sind.Len() == 0 {
			return cnt, ErrArgs
		}
	default:
		return cnt, ErrArgs
	}
	sb := strings.Builder{}
	for i := 0; i < sind.Len(); i++ {
		ind := sind.Index(i).Interface()
		mi, ok := defaultmodelCache.getByMd(ind)
		if !ok {
			return cnt, ErrNotModel
		}
		docId := mi.getPKValue(ind)
		if docId == "0" {
			id := time.Now().UnixMilli()
			docId = fmt.Sprintf("%v", id)
			mi.setFieldValue(ind, mi.pkkey, id)
		}
		b, err := json.Marshal(ind)
		if err != nil {
			return cnt, err
		}
		sb.WriteString(`{ "create": { "_index" : "` + mi.indexName + `", "_id": "` + docId + `"} }` + "\n")
		sb.Write(b)
		sb.WriteString("\n")

		cnt++
	}
	return cnt, es.BulkDoc([]byte(sb.String()))
}

func (o *defaultOrm) Update(md interface{}, cols ...string) (int64, error) {
	mi, ok := defaultmodelCache.getByMd(md)
	if !ok {
		return 0, fmt.Errorf("model not exist %d", md)
	}
	if len(cols) == 0 {
		cols = mi.fieldNames
	}
	f := appendField(mi, md, cols...)
	docId := mi.getPKValue(md)
	return 1, es.UpdateDoc(mi.indexName, docId, f)
}

func (o *defaultOrm) Delete(md interface{}, cols ...string) (int64, error) {
	mi, ok := defaultmodelCache.getByMd(md)
	if !ok {
		return 0, fmt.Errorf("model not exist %d", md)
	}
	if len(cols) == 0 {
		docId := mi.getPKValue(md)
		err := es.DeleteDoc(mi.indexName, docId)
		if err != nil {
			return 0, err
		}
	} else {
		f := appendField(mi, md, cols...)
		filter := es.AppendFilter(f)
		err := es.DeleteByQuery(mi.indexName, filter)
		if err != nil {
			return 0, err
		}
	}
	return 1, nil
}

func (o *defaultOrm) Read(md interface{}, cols ...string) error {
	mi, ok := defaultmodelCache.getByMd(md)
	if !ok {
		return fmt.Errorf("model not exist %d", md)
	}
	if len(cols) == 0 {
		cols = []string{mi.pkkey}
	}
	f := appendField(mi, md, cols...)
	filter := es.AppendFilter(f)
	query := es.Query{From: 0, Size: 1, Filter: filter}
	_, list, err := es.FilterSearch[map[string]interface{}](mi.indexName, query)
	if err != nil {
		return err
	}
	if len(list) > 0 {
		jsonstring, err := json.Marshal(list[0])
		if err != nil {
			return err
		}
		err = json.Unmarshal(jsonstring, md)
		return err
	}
	return ErrNoRows
}

func appendField(mi *modelInfo, md interface{}, cols ...string) map[string]interface{} {
	f := map[string]interface{}{}
	for _, fieldName := range cols {
		f[FirstLower(fieldName)] = mi.getFieldValue(md, fieldName)
	}
	return f
}
