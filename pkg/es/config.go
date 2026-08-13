package es

import (
	"fmt"
	"time"

	logs "go-iot/pkg/logger"
	"go-iot/pkg/option"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// the config of elasticsearch
type EsConfig struct {
	Url              string // es url默认http://localhost:9200
	Username         string
	Password         string
	BufferSize       int    // 缓冲大小 默认10000
	BulkSize         int    // 每次批量提交数 默认2000
	WarnTime         int    // warn日志时间当保存时间操作指定时间时输出日志，默认1000ms
	NumberOfShards   string // 分片数，默认1
	NumberOfReplicas string // 副本数，默认0
	// RetentionMonths 时序保留月数，0=不按全局清理
	RetentionMonths int
	// RetentionCheckHours 清理间隔小时，0=不启动任务
	RetentionCheckHours int
}

func (r EsConfig) String() string {
	return fmt.Sprintf("[url=%s, username=%v, BufferSize=%v, BulkSize=%v, WarnTime=%v, NumberOfShards=%s, NumberOfReplicas=%s, RetentionMonths=%d, RetentionCheckHours=%d]",
		r.Url, r.Username, r.BufferSize, r.BulkSize,
		r.WarnTime, r.NumberOfShards, r.NumberOfReplicas, r.RetentionMonths, r.RetentionCheckHours)
}

var DefaultEsConfig EsConfig = EsConfig{
	Url:                 "http://localhost:9200",
	BufferSize:          10000,
	BulkSize:            2000,
	WarnTime:            1000,
	NumberOfShards:      "1",
	NumberOfReplicas:    "0",
	RetentionMonths:     0,
	RetentionCheckHours: 24,
}

func Config(opt *option.Options) {
	DefaultEsConfig.Url = opt.Es.Url
	DefaultEsConfig.Username = opt.Es.Username
	DefaultEsConfig.Password = opt.Es.Password
	DefaultEsConfig.NumberOfShards = opt.Es.NumberOfShards
	DefaultEsConfig.NumberOfReplicas = opt.Es.NumberOfReplicas
	DefaultEsConfig.BufferSize = opt.Es.BufferSize
	DefaultEsConfig.WarnTime = opt.Es.WarnTime
	DefaultEsConfig.RetentionMonths = opt.Es.RetentionMonths
	if DefaultEsConfig.RetentionMonths < 0 {
		DefaultEsConfig.RetentionMonths = 0
	}
	// flag 默认 24；yaml/flag 设为 0 表示关闭定时清理任务
	DefaultEsConfig.RetentionCheckHours = opt.Es.RetentionCheckHours
	if DefaultEsConfig.RetentionCheckHours < 0 {
		DefaultEsConfig.RetentionCheckHours = 0
	}
	logs.Infof("elasticsearch config: %v", DefaultEsConfig)
	for {
		var client, err = getEsClient()
		if err == nil {
			var resp *esapi.Response
			resp, err = client.Cat.Health()
			if err == nil {
				logs.Infof("%s", resp.String())
				break
			}
		}
		logs.Errorf("elasticsearch error: %v", err)
		time.Sleep(5 * time.Second) // 等待5秒后重试
	}
}
