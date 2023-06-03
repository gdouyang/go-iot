package es

import (
	"encoding/json"
	"errors"
)

const Prefix = "goiot-"

const DefaultDateFormat string = "yyyy-MM||yyyy-MM-dd||yyyy-MM-dd HH:mm:ss||yyyy-MM-dd HH:mm:ss.SSS||epoch_millis"

type Property struct {
	Type        string `json:"type"`
	IgnoreAbove string `json:"ignore_above,omitempty"`
	Format      string `json:"format,omitempty"`
}

const (
	IN   = "IN"
	EQ   = "EQ"   // Equal to
	GT   = "GT"   // Greater than
	GTE  = "GTE"  // Greater than or Equal
	LT   = "LT"   // less then
	LTE  = "LTE"  // less then or Equal
	LIKE = "LIKE" // like
	BTW  = "BTW"  // between
)

type SearchTerm struct {
	Key   string `json:"key"`   // 查询的字段
	Value any    `json:"value"` // 值
	Oper  string `json:"oper"`  // 操作符IN,EQ,GT,LE,LIKE;默认(EQ)
}

type Query struct {
	From        int
	Size        int
	Filter      []map[string]any
	Sort        []map[string]SortOrder
	Includes    []string
	SearchAfter []any
}

type SortOrder struct {
	Order string `json:"order"` // desc, asc
}

type IndexResponse struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Result  string
}

type SearchResponse struct {
	Total       int64
	Sources     []byte
	FirstSource []byte
	LastSort    []any
}

func (r *SearchResponse) ConvertSource(result any) error {
	err := json.Unmarshal(r.Sources, result)
	return err
}

func NewEsError(e error) *ErrorResponse {
	return &ErrorResponse{Info: &ErrorInfo{Reason: e.Error()}}
}

type ErrorResponse struct {
	Info *ErrorInfo `json:"error,omitempty"`
}

func (e *ErrorResponse) Error() error {
	return errors.New(e.Info.Reason)
}

func (e *ErrorResponse) Is404() bool {
	return e.Info.Reason == Err404.Error()
}

type ErrorInfo struct {
	RootCause []*ErrorInfo
	Type      string
	Reason    string
	Phase     string
}
