package timeseries

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"go-iot/pkg/option"

	logs "go-iot/pkg/logger"
)

// TdengineConfig TDengine REST 与批量写入运行时配置。
type TdengineConfig struct {
	Enabled         bool
	Url             string
	Username        string
	Password        string
	Database        string
	Timeout         time.Duration
	BufferSize      int
	BulkSize        int
	FlushInterval   time.Duration
	basicAuthHeader string // "Basic xxx"
}

func (c TdengineConfig) String() string {
	return fmt.Sprintf("[enabled=%v, url=%s, user=%s, database=%s, timeout=%s, buffer=%d, bulk=%d, flush=%s]",
		c.Enabled, c.Url, c.Username, c.Database, c.Timeout, c.BufferSize, c.BulkSize, c.FlushInterval)
}

// DefaultTdengineConfig 默认连接本地 Docker 常见端口（默认不启用，需配置 tdengine.enabled）。
var DefaultTdengineConfig = TdengineConfig{
	Enabled:       false,
	Url:           "http://localhost:6041",
	Username:      "root",
	Password:      "taosdata",
	Database:      "goiot",
	Timeout:       30 * time.Second,
	BufferSize:    20000,
	BulkSize:      500,
	FlushInterval: time.Second,
}

var (
	tdCfgMu sync.RWMutex
	tdCfg   = DefaultTdengineConfig
)

// ConfigTdengine 从 option 加载 TDengine 配置（app 启动时调用）。
func ConfigTdengine(opt *option.Options) {
	if opt == nil {
		return
	}
	cfg := DefaultTdengineConfig
	cfg.Enabled = opt.Tdengine.Enabled
	if u := strings.TrimSpace(opt.Tdengine.Url); u != "" {
		cfg.Url = strings.TrimRight(u, "/")
	}
	if opt.Tdengine.Username != "" {
		cfg.Username = opt.Tdengine.Username
	}
	if opt.Tdengine.Password != "" {
		cfg.Password = opt.Tdengine.Password
	}
	if d := strings.TrimSpace(opt.Tdengine.Database); d != "" {
		cfg.Database = d
	}
	if opt.Tdengine.TimeoutSeconds > 0 {
		cfg.Timeout = time.Duration(opt.Tdengine.TimeoutSeconds) * time.Second
	}
	if opt.Tdengine.BufferSize > 0 {
		cfg.BufferSize = opt.Tdengine.BufferSize
	}
	if opt.Tdengine.BulkSize > 0 {
		cfg.BulkSize = opt.Tdengine.BulkSize
	}
	if opt.Tdengine.FlushIntervalMs > 0 {
		cfg.FlushInterval = time.Duration(opt.Tdengine.FlushIntervalMs) * time.Millisecond
	}
	cfg.basicAuthHeader = "Basic " + base64.StdEncoding.EncodeToString(
		[]byte(cfg.Username+":"+cfg.Password))

	tdCfgMu.Lock()
	tdCfg = cfg
	tdCfgMu.Unlock()

	// 更新批量写入参数（ch 容量取 buffer-size，启动早期无并发，重建安全）
	defaultTdBatch.applyConfig(cfg.BufferSize, cfg.BulkSize, cfg.FlushInterval)

	logs.Infof("tdengine config: %v", cfg)
}

// TdengineEnabled 运行时是否启用 TDengine 作为可选时序策略。
func TdengineEnabled() bool {
	return getTdConfig().Enabled
}

func getTdConfig() TdengineConfig {
	tdCfgMu.RLock()
	defer tdCfgMu.RUnlock()
	return tdCfg
}

// tdRestSQLURL 完整 REST SQL 地址。
func tdRestSQLURL() string {
	c := getTdConfig()
	base := c.Url
	if base == "" {
		base = "http://localhost:6041"
	}
	// 已含 rest/sql 则直接用
	if strings.Contains(base, "/rest/sql") {
		return base
	}
	u, err := url.Parse(base)
	if err != nil {
		return base + "/rest/sql"
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/rest/sql"
	return u.String()
}

func tdDatabase() string {
	c := getTdConfig()
	if c.Database == "" {
		return "goiot"
	}
	return c.Database
}

func tdAuthHeader() string {
	c := getTdConfig()
	if c.basicAuthHeader != "" {
		return c.basicAuthHeader
	}
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.Username+":"+c.Password))
}

func tdHTTPTimeout() time.Duration {
	c := getTdConfig()
	if c.Timeout <= 0 {
		return 30 * time.Second
	}
	return c.Timeout
}
