package es

import "fmt"

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

const DefaultDateFormat string = "yyyy-MM||yyyy-MM-dd||yyyy-MM-dd HH:mm:ss||yyyy-MM-dd HH:mm:ss.SSS||epoch_millis"
