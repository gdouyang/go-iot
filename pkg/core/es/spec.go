package es

import (
	"encoding/json"
)

const Prefix = "goiot-"

const DefaultDateFormat string = "yyyy-MM||yyyy-MM-dd||yyyy-MM-dd HH:mm:ss||yyyy-MM-dd HH:mm:ss.SSS||epoch_millis"

type Property struct {
	Type        string `json:"type"`
	IgnoreAbove string `json:"ignore_above,omitempty"`
	Format      string `json:"format,omitempty"`
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

type EsResponse struct {
	Data       string
	StatusCode int
	IsError    bool
}

func (e *EsResponse) Is404() bool {
	return e.StatusCode == 404
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
