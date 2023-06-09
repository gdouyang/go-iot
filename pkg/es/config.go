package es

import (
	"fmt"
	"strconv"

	logs "go-iot/pkg/logger"
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
			logs.Errorf("es.buffersize error: %v", err)
		}
	})
	fn("es.warntime", func(s string) {
		num, err := strconv.Atoi(s)
		if err == nil {
			DefaultEsConfig.WarnTime = num
		} else {
			logs.Errorf("es.warntime error: %v", err)
		}
	})
	logs.Infof("es config: %v", DefaultEsConfig)
}
