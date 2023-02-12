package es

import (
	"fmt"
	"strconv"

	"github.com/beego/beego/v2/core/logs"
)

// the config of elasticsearch
type EsConfig struct {
	Url              string
	Username         string
	Password         string
	BufferSize       int    // default 10000
	BulkSize         int    // default 5000
	WarnTime         int    // warn日志时间当保存时间操作指定时间时输出日志，默认1000ms
	NumberOfShards   string // 分片数
	NumberOfReplicas string // 副本数
}

func (r EsConfig) String() string {
	return fmt.Sprintf("[url=%s, username=%v, BufferSize=%v, BulkSize=%v, WarnTime=%v, NumberOfShards=%s, NumberOfReplicas=%s]",
		r.Url, r.Username, r.BufferSize, r.BulkSize,
		r.WarnTime, r.NumberOfShards, r.NumberOfReplicas)
}

var DefaultEsConfig EsConfig = EsConfig{
	Url:              "http://localhost:9200",
	BufferSize:       10000,
	BulkSize:         5000,
	WarnTime:         1000,
	NumberOfShards:   "1",
	NumberOfReplicas: "0",
}

func Config(fn func(key string, call func(string))) {
	fn("es.url", func(s string) {
		DefaultEsConfig.Url = s
	})
	fn("es.usename", func(s string) {
		DefaultEsConfig.Username = s
	})
	fn("es.password", func(s string) {
		DefaultEsConfig.Password = s
	})
	fn("es.numberOfShards", func(s string) {
		DefaultEsConfig.NumberOfShards = s
	})
	fn("es.numberOfReplicas", func(s string) {
		DefaultEsConfig.NumberOfReplicas = s
	})
	fn("es.buffersize", func(s string) {
		num, err := strconv.Atoi(s)
		if err == nil {
			DefaultEsConfig.BufferSize = num
		} else {
			logs.Error("es.buffersize error:", err)
		}
	})
	fn("es.warntime", func(s string) {
		num, err := strconv.Atoi(s)
		if err == nil {
			DefaultEsConfig.WarnTime = num
		} else {
			logs.Error("es.warntime error:", err)
		}
	})
	logs.Info("es config: ", DefaultEsConfig)
}

const DefaultDateFormat string = "yyyy-MM||yyyy-MM-dd||yyyy-MM-dd HH:mm:ss||yyyy-MM-dd HH:mm:ss.SSS||epoch_millis"

type EsQueryResult[T any] struct {
	Hits EsHit[T] `json:"hits"`
}

type EsHit[T any] struct {
	Total HitTotal    `json:"total"`
	Hits  []EsHits[T] `json:"hits"`
}

type HitTotal struct {
	Value int `json:"value"`
}

type EsHits[T any] struct {
	Index  string `json:"_index"`
	ID     string `json:"_id"`
	Source T      `json:"_source"`
}

type EsErrorResult struct {
	Status    int     `json:"status"`
	Error     EsError `json:"error"`
	OriginErr error   `json:"-"`
}

func NewEsError(originerr error) *EsErrorResult {
	return &EsErrorResult{OriginErr: originerr}
}

type EsError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
	Index  string `json:"index"`
}
