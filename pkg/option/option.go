package option

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type Cluster struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"` // 是否启用集群（配置/文档统一用 enabled）
	Url     string `yaml:"url"`
	Token   string `yaml:"token"`
	Index   int    `yaml:"index"`
	Hosts   string `yaml:"hosts"`
}

type Es struct {
	Url              string `yaml:"url"`
	Username         string `yaml:"usename"`
	Password         string `yaml:"password"`
	NumberOfShards   string `yaml:"numberOfShards"`
	NumberOfReplicas string `yaml:"numberOfReplicas"`
	BufferSize       int    `yaml:"bufferSize"`
	BulkSize         int    `yaml:"bulkSize"`
	WarnTime         int    `yaml:"warntime"`
	// RetentionMonths 时序最大保存月数（properties/event/logs）。0=默认不清理（升级安全）。
	RetentionMonths int `yaml:"retention-months"`
	// RetentionCheckHours 清理任务间隔（小时）。0=不启动定时任务。默认 24。
	RetentionCheckHours int `yaml:"retention-check-hours"`
}

type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	Db       int    `yaml:"db"`
	PoolSize int    `yaml:"pool-size"` // 连接池大小；≤0 时用 pkg/redis 默认值(100)
}

type Log struct {
	Dir    string `yaml:"filename"`
	Format string `yaml:"format"`
	Level  string `yaml:"level"`
	// ReloadInterval 日志级别热刷新间隔（秒）。
	// 默认 30；设为 0 关闭定时从配置文件刷新。
	ReloadInterval int `yaml:"reload-interval"`
}

type Certificate struct {
	Name string `yaml:"name"`
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}
type Mqtt struct {
	Host        string        `yaml:"host"`
	Name        string        `yaml:"name"`
	Port        int           `yaml:"port"`
	UseTLS      bool          `yaml:"useTLS"`
	Certificate []Certificate `yaml:"certificate"`
}

// Admin 首次初始化默认管理员相关配置。
// 仅在创建 id=1 的 admin 时使用；库中已有用户时不会改密。
type Admin struct {
	// Password 初始密码，未配置时默认 123456
	Password string `yaml:"password"`
}

// Script 编解码脚本宿主能力开关。
type Script struct {
	// HTTPEnabled 是否允许 globe.HttpRequest / HttpRequestAsync。默认 true。
	HTTPEnabled bool `yaml:"http-enabled"`
	// HTTPBlockPrivate 是否禁止访问私网/本机地址（SSRF 基线）。默认 true。
	HTTPBlockPrivate bool `yaml:"http-block-private"`
	// VMPoolSize 每个产品编解码 JS 引擎池大小。默认 20，上限 MaxScriptVMPoolSize。
	VMPoolSize int `yaml:"vm-pool-size"`
}

// Rule 规则引擎相关配置。
type Rule struct {
	// MaxShakeLimitTime 防抖/抖动限制时间轮最大秒数，默认 3600。
	MaxShakeLimitTime int `yaml:"max-shake-limit-time"`
}

// Tdengine 时序存储 storePolicy=tdengine（REST SQL）。
type Tdengine struct {
	// Enabled 是否启用 TDengine 作为可选时序策略（关闭后新建产品不可选）
	Enabled bool `yaml:"enabled"`
	// Url 服务根地址，如 http://localhost:6041（自动拼 /rest/sql）
	Url string `yaml:"url"`
	// Username REST Basic 用户，默认 root
	Username string `yaml:"username"`
	// Password REST Basic 密码，默认 taosdata
	Password string `yaml:"password"`
	// Database 业务库名，默认 goiot
	Database string `yaml:"database"`
	// TimeoutSeconds HTTP 超时秒数，默认 30
	TimeoutSeconds int `yaml:"timeout-seconds"`
	// BufferSize 批量写入队列容量，默认 20000
	BufferSize int `yaml:"buffer-size"`
	// BulkSize 单次批量提交行数，默认 500
	BulkSize int `yaml:"bulk-size"`
	// FlushIntervalMs 定时 flush 间隔毫秒，默认 1000
	FlushIntervalMs int `yaml:"flush-interval-ms"`
}

// DefaultAdminPassword 配置未写 password 时的默认初始密码。
const DefaultAdminPassword = "123456"

const (
	// DefaultScriptVMPoolSize 每个产品默认 JS 引擎数。
	DefaultScriptVMPoolSize = 20
	// MaxScriptVMPoolSize 防止误配把内存打满。
	MaxScriptVMPoolSize = 500
)

// Options is the start-up options.
type Options struct {
	flags   *pflag.FlagSet
	viper   *viper.Viper
	yamlStr string

	// Flags from command line only.
	ShowConfig bool   `yaml:"-"`
	ConfigFile string `yaml:"-"`

	// meta
	APIAddr string `yaml:"api-addr"`

	// cluster options
	Cluster Cluster `yaml:"cluster"`

	// es配置
	Es Es `yaml:"es"`

	// TDengine 时序（storePolicy=tdengine）
	Tdengine Tdengine `yaml:"tdengine"`

	// redis配置
	Redis Redis `yaml:"redis"`

	// 日志配置
	Log Log `yaml:"logs"`

	// Mqtt配置
	Mqtt Mqtt `yaml:"mqtt"`

	// 默认管理员（仅首次创建时生效）
	Admin Admin `yaml:"admin"`

	// 脚本宿主安全
	Script Script `yaml:"script"`

	// 规则引擎
	Rule Rule `yaml:"rule"`

	// 控制台输出的banner
	Banner string `yaml:"banner"`
}

// AdminPassword 返回初始管理员密码（空则 DefaultAdminPassword）。
func (opt *Options) AdminPassword() string {
	if opt == nil || len(opt.Admin.Password) == 0 {
		return DefaultAdminPassword
	}
	return opt.Admin.Password
}

// ScriptHTTPEnabled 脚本是否允许 HTTP 出站（Global 未初始化时默认 true）。
func ScriptHTTPEnabled() bool {
	if Global == nil {
		return true
	}
	return Global.Script.HTTPEnabled
}

// ScriptHTTPBlockPrivate 脚本 HTTP 是否拦截私网（Global 未初始化时默认 true）。
func ScriptHTTPBlockPrivate() bool {
	if Global == nil {
		return true
	}
	return Global.Script.HTTPBlockPrivate
}

// ScriptVMPoolSize 每个产品编解码 VM 池大小（Global 未初始化或非法值时默认 20）。
func ScriptVMPoolSize() int {
	if Global == nil {
		return DefaultScriptVMPoolSize
	}
	return clampScriptVMPoolSize(Global.Script.VMPoolSize)
}

func clampScriptVMPoolSize(n int) int {
	if n <= 0 {
		return DefaultScriptVMPoolSize
	}
	if n > MaxScriptVMPoolSize {
		return MaxScriptVMPoolSize
	}
	return n
}

// TdengineEnabled 是否启用 TDengine 时序选项。
func TdengineEnabled() bool {
	if Global == nil {
		return false
	}
	return Global.Tdengine.Enabled
}

const banner string = `
               ╔══╗     ╔╗   
               ╚╣╠╝    ╔╝╚╗ 
  ╔══╗╔══╗      ║║ ╔══╗╚╗╔╝ 
  ║╔╗║║╔╗║╔═══╗ ║║ ║╔╗║ ║║  
  ║╚╝║║╚╝║╚═══╝╔╣╠╗║╚╝║ ║╚╗ 
  ╚═╗║╚══╝     ╚══╝╚══╝ ╚═╝ 
  ╔═╝║                      
  ╚══╝                      
═════════════════════════════
release: %s, build_time: %s, commit: %s, repo: %s
═════════════════════════════
`

var Global *Options

// New creates a default Options.
func New() *Options {
	opt := &Options{
		flags: pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError),
		viper: viper.New(),
	}

	opt.flags.StringVar(&opt.APIAddr, "api-addr", ":8088", "api服务地址([host]:port)")
	opt.flags.StringVarP(&opt.ConfigFile, "config-file", "f", "conf/app.yaml", "配置文件位置")
	// 日志配置
	opt.flags.StringVar(&opt.Log.Dir, "logs.filename", "logs/goiot.log", "日志存放位置")
	opt.flags.StringVar(&opt.Log.Format, "logs.format", "text", "日志格式(text, json)")
	opt.flags.StringVar(&opt.Log.Level, "logs.level", "info", "日志级别(debug,info,warn,error)")
	opt.flags.IntVar(&opt.Log.ReloadInterval, "logs.reload-interval", 30, "日志级别热刷新间隔秒数，0关闭")
	// Mqtt配置
	opt.flags.IntVar(&opt.Mqtt.Port, "mqtt.port", 1883, "mqtt服务端口")
	// Redis配置
	opt.flags.StringVar(&opt.Redis.Addr, "redis.addr", "localhost:6379", "redis地址(localhost:6379)")
	opt.flags.StringVar(&opt.Redis.Password, "redis.password", "", "redis密码")
	opt.flags.IntVar(&opt.Redis.Db, "redis.db", 0, "redis数据库")
	opt.flags.IntVar(&opt.Redis.PoolSize, "redis.pool-size", 100, "redis连接池大小(默认100，大并发上线场景建议调大)")
	// ES配置
	opt.flags.StringVar(&opt.Es.Url, "es.url", "http://localhost:9200", "elasticsearch地址(http://localhost:9200)")
	opt.flags.StringVar(&opt.Es.Username, "es.usename", "", "elasticsearch用户名")
	opt.flags.StringVar(&opt.Es.Password, "es.password", "", "elasticsearch密码")
	opt.flags.StringVar(&opt.Es.NumberOfShards, "es.numberOfShards", "1", "时序数据分片数")
	opt.flags.StringVar(&opt.Es.NumberOfReplicas, "es.numberOfReplicas", "0", "数序数据副本数")
	opt.flags.IntVar(&opt.Es.BufferSize, "es.bufferSize", 10000, "时序数据内存缓冲大小")
	opt.flags.IntVar(&opt.Es.BulkSize, "es.bulkSize", 1000, "时序数据批量提交大小")
	opt.flags.IntVar(&opt.Es.WarnTime, "es.warntime", 1000, "时序数据保存时间阈值")
	opt.flags.IntVar(&opt.Es.RetentionMonths, "es.retention-months", 0, "ES时序最大保存月数，0关闭清理")
	opt.flags.IntVar(&opt.Es.RetentionCheckHours, "es.retention-check-hours", 24, "ES时序清理任务间隔小时，0不启动")
	// TDengine
	opt.flags.BoolVar(&opt.Tdengine.Enabled, "tdengine.enabled", false, "是否启用 TDengine 时序存储选项")
	opt.flags.StringVar(&opt.Tdengine.Url, "tdengine.url", "http://localhost:6041", "TDengine REST 根地址")
	opt.flags.StringVar(&opt.Tdengine.Username, "tdengine.username", "root", "TDengine 用户名")
	opt.flags.StringVar(&opt.Tdengine.Password, "tdengine.password", "taosdata", "TDengine 密码")
	opt.flags.StringVar(&opt.Tdengine.Database, "tdengine.database", "goiot", "TDengine 业务库名")
	opt.flags.IntVar(&opt.Tdengine.TimeoutSeconds, "tdengine.timeout-seconds", 30, "TDengine HTTP 超时秒")
	opt.flags.IntVar(&opt.Tdengine.BufferSize, "tdengine.buffer-size", 20000, "TDengine 批量写队列容量")
	opt.flags.IntVar(&opt.Tdengine.BulkSize, "tdengine.bulk-size", 500, "TDengine 单次批量行数")
	opt.flags.IntVar(&opt.Tdengine.FlushIntervalMs, "tdengine.flush-interval-ms", 1000, "TDengine 定时 flush 毫秒")
	// 集群配置
	opt.flags.BoolVar(&opt.Cluster.Enabled, "cluster.enabled", false, "是否启用集群")
	opt.flags.StringVar(&opt.Cluster.Name, "cluster.name", "", "集群节点名")
	opt.flags.IntVar(&opt.Cluster.Index, "cluster.index", 1, "集群index")
	opt.flags.StringVar(&opt.Cluster.Token, "cluster.token", "", "集群通讯token")
	opt.flags.StringVar(&opt.Cluster.Url, "cluster.url", "", "本机url")
	opt.flags.StringVar(&opt.Cluster.Hosts, "cluster.hosts", "", "集群内主机列表")

	// 默认管理员初始密码（仅首次创建 admin 时使用）
	opt.flags.StringVar(&opt.Admin.Password, "admin.password", DefaultAdminPassword, "默认管理员初始密码（仅库中无 admin 时生效；生产请改）")

	// 脚本 HTTP / SSRF 基线
	opt.flags.BoolVar(&opt.Script.HTTPEnabled, "script.http-enabled", true, "是否允许编解码脚本 HttpRequest")
	opt.flags.BoolVar(&opt.Script.HTTPBlockPrivate, "script.http-block-private", true, "脚本 HTTP 是否禁止访问私网/本机（SSRF）")
	opt.flags.IntVar(&opt.Script.VMPoolSize, "script.vm-pool-size", DefaultScriptVMPoolSize, "每个产品编解码 JS 引擎池大小")

	// 规则引擎
	opt.flags.IntVar(&opt.Rule.MaxShakeLimitTime, "rule.max-shake-limit-time", 3600, "规则引擎防抖时间轮最大秒数")
	opt.flags.StringVar(&opt.Banner, "banner", banner, "")
	opt.viper.BindPFlags(opt.flags)

	return opt
}

// YAML returns yaml string of option, need to be called after calling Parse.
func (opt *Options) YAML() string {
	return opt.yamlStr
}

// Parse parses all arguments, returns normal message without error if --help/--version set.
func (opt *Options) Parse() (string, error) {
	err := opt.flags.Parse(os.Args[1:])
	if err != nil {
		return "", err
	}

	opt.viper.AutomaticEnv()
	opt.viper.SetEnvPrefix("GOIOT")
	opt.viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	if opt.ConfigFile != "" {
		opt.viper.SetConfigFile(opt.ConfigFile)
		opt.viper.SetConfigType("yaml")
		err := opt.viper.ReadInConfig()
		if err != nil {
			return "", fmt.Errorf("read config file %s failed: %v",
				opt.ConfigFile, err)
		}
	}

	// NOTE: Workaround because viper does not treat env vars the same as other config.
	// Reference: https://github.com/spf13/viper/issues/188#issuecomment-399518663
	for _, key := range opt.viper.AllKeys() {
		val := opt.viper.Get(key)
		opt.viper.Set(key, val)
	}

	err = opt.viper.Unmarshal(opt, func(c *mapstructure.DecoderConfig) {
		c.TagName = "yaml"
	})
	if err != nil {
		return "", fmt.Errorf("yaml file unmarshal failed, please make sure you provide valid yaml file, %v", err)
	}

	// 兼容旧配置键 cluster.enable（未写 enabled 时）
	if !opt.viper.IsSet("cluster.enabled") && opt.viper.IsSet("cluster.enable") {
		opt.Cluster.Enabled = opt.viper.GetBool("cluster.enable")
	}

	// 兼容旧顶层键 max-shake-limit-time → rule.max-shake-limit-time
	if !opt.viper.IsSet("rule.max-shake-limit-time") && opt.viper.IsSet("max-shake-limit-time") {
		opt.Rule.MaxShakeLimitTime = opt.viper.GetInt("max-shake-limit-time")
	}
	if opt.Rule.MaxShakeLimitTime <= 0 {
		opt.Rule.MaxShakeLimitTime = 3600
	}

	// tdengine 默认值（未写或空串时）
	// enabled 默认 false；仅当 flag/yaml 显式设置时为 true（BindPFlags 默认 false）
	if strings.TrimSpace(opt.Tdengine.Url) == "" {
		opt.Tdengine.Url = "http://localhost:6041"
	}
	if strings.TrimSpace(opt.Tdengine.Username) == "" {
		opt.Tdengine.Username = "root"
	}
	if strings.TrimSpace(opt.Tdengine.Password) == "" {
		opt.Tdengine.Password = "taosdata"
	}
	if strings.TrimSpace(opt.Tdengine.Database) == "" {
		opt.Tdengine.Database = "goiot"
	}
	if opt.Tdengine.TimeoutSeconds <= 0 {
		opt.Tdengine.TimeoutSeconds = 30
	}
	if opt.Tdengine.BufferSize <= 0 {
		opt.Tdengine.BufferSize = 20000
	}
	if opt.Tdengine.BulkSize <= 0 {
		opt.Tdengine.BulkSize = 500
	}
	if opt.Tdengine.FlushIntervalMs <= 0 {
		opt.Tdengine.FlushIntervalMs = 1000
	}

	// script 布尔默认：未配置时保持安全默认（HTTP 开、拦私网）
	// viper/Unmarshal 对 bool 零值会变成 false，需用 IsSet 区分「显式 false」与「未写」。
	if !opt.viper.IsSet("script.http-enabled") {
		opt.Script.HTTPEnabled = true
	}
	if !opt.viper.IsSet("script.http-block-private") {
		opt.Script.HTTPBlockPrivate = true
	}
	opt.Script.VMPoolSize = clampScriptVMPoolSize(opt.Script.VMPoolSize)

	buff, err := yaml.Marshal(opt)
	if err != nil {
		return "", fmt.Errorf("marshal config to yaml failed: %v", err)
	}
	opt.yamlStr = string(buff)

	if opt.ShowConfig {
		fmt.Printf("%s", opt.yamlStr)
	}

	Global = opt

	return "", nil
}

// ReadLogLevelFromFile 仅从配置文件读取 logs.level，不改动其它运行时配置。
// 文件未配置 level 时返回 "info"。
func (opt *Options) ReadLogLevelFromFile() (string, error) {
	if opt == nil || opt.ConfigFile == "" {
		return "", fmt.Errorf("config file not set")
	}
	data, err := os.ReadFile(opt.ConfigFile)
	if err != nil {
		return "", fmt.Errorf("read config file %s: %w", opt.ConfigFile, err)
	}
	var partial struct {
		Logs struct {
			Level string `yaml:"level"`
		} `yaml:"logs"`
	}
	if err := yaml.Unmarshal(data, &partial); err != nil {
		return "", fmt.Errorf("parse config file %s: %w", opt.ConfigFile, err)
	}
	level := strings.ToLower(strings.TrimSpace(partial.Logs.Level))
	if level == "" {
		level = "info"
	}
	return level, nil
}
