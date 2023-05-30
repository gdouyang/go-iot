package es

import (
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
	Key   string
	Value interface{}
	Oper  string // IN,EQ,GT,LE,LIKE
}

type Query struct {
	From        int
	Size        int
	Filter      []map[string]interface{}
	Sort        []map[string]SortOrder
	Includes    []string
	SearchAfter []string
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

type SearchResponse[T any] struct {
	Took int64
	Hits struct {
		Total struct {
			Value int64
		}
		Hits []*SearchHit[T]
	}
}

type SearchHit[T any] struct {
	Score   float64 `json:"_score"`
	Index   string  `json:"_index"`
	Type    string  `json:"_type"`
	Version int64   `json:"_version,omitempty"`

	Source T `json:"_source"`
	Sort   []interface{}
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

type ErrorInfo struct {
	RootCause []*ErrorInfo
	Type      string
	Reason    string
	Phase     string
}
