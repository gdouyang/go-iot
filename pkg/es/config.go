package es

import (
	"fmt"

	logs "go-iot/pkg/logger"
	"go-iot/pkg/option"
)

// the config of elasticsearch
type EsConfig struct {
	Url              string
	Username         string
	Password         string
	BufferSize       int    // 缓冲大小 10000
	BulkSize         int    // 每次批量提交数 1000
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
	BulkSize:         1000,
	WarnTime:         1000,
	NumberOfShards:   "1",
	NumberOfReplicas: "0",
}

func Config(opt *option.Options) {
	DefaultEsConfig.Url = opt.Es.Url
	DefaultEsConfig.Username = opt.Es.Username
	DefaultEsConfig.Password = opt.Es.Password
	DefaultEsConfig.NumberOfShards = opt.Es.NumberOfShards
	DefaultEsConfig.NumberOfReplicas = opt.Es.NumberOfReplicas
	DefaultEsConfig.BufferSize = opt.Es.BufferSize
	DefaultEsConfig.WarnTime = opt.Es.WarnTime
	logs.Infof("es config: %v", DefaultEsConfig)
}
