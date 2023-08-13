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
	Name   string `yaml:"name"`
	Enable bool   `yaml:"enable"`
	Url    string `yaml:"url"`
	Token  string `yaml:"token"`
	Index  int    `yaml:"index"`
	Hosts  string `yaml:"hosts"`
}

type Es struct {
	Url              string `yaml:"url"`
	Username         string `yaml:"usename"`
	Password         string `yaml:"password"`
	NumberOfShards   string `yaml:"numberOfShards"`
	NumberOfReplicas string `yaml:"numberOfReplicas"`
	BufferSize       int    `yaml:"bulksize"`
	WarnTime         int    `yaml:"warntime"`
}

type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	Db       int    `yaml:"db"`
}

type Log struct {
	Dir    string `yaml:"filename"`
	Format string `yaml:"format"`
	Level  string `yaml:"level"`
}

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

	// es options
	Es Es `yaml:"es"`

	// redis options
	Redis Redis `yaml:"redis"`

	// Path.
	Log Log `yaml:"logs"`
}

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
	// Redis配置
	opt.flags.StringVar(&opt.Redis.Addr, "redis.addr", "localhost:6379", "redis地址(localhost:6379)")
	opt.flags.StringVar(&opt.Redis.Password, "redis.password", "", "redis密码")
	opt.flags.IntVar(&opt.Redis.Db, "redis.db", 0, "redis数据库")
	// ES配置
	opt.flags.StringVar(&opt.Es.Url, "es.url", "http://localhost:9200", "elasticsearch地址(http://localhost:9200)")
	opt.flags.StringVar(&opt.Es.Username, "es.usename", "", "elasticsearch用户名")
	opt.flags.StringVar(&opt.Es.Password, "es.password", "", "elasticsearch密码")
	opt.flags.StringVar(&opt.Es.NumberOfShards, "es.numberOfShards", "1", "时序数据分片数")
	opt.flags.StringVar(&opt.Es.NumberOfReplicas, "es.numberOfReplicas", "0", "数序数据副本数")
	opt.flags.IntVar(&opt.Es.BufferSize, "es.buffersize", 10000, "时序数据库批量提交buffer")
	opt.flags.IntVar(&opt.Es.WarnTime, "es.warntime", 1000, "时序数据库保存时间阈值")
	// 集群配置
	opt.flags.BoolVar(&opt.Cluster.Enable, "cluster.enabled", false, "是否启用集群")
	opt.flags.StringVar(&opt.Cluster.Name, "cluster.name", "node", "集群节点名")
	opt.flags.IntVar(&opt.Cluster.Index, "cluster.index", 1, "集群index")
	opt.flags.StringVar(&opt.Cluster.Token, "cluster.token", "", "集群通讯token")
	opt.flags.StringVar(&opt.Cluster.Url, "cluster.url", "", "本机url")
	opt.flags.StringVar(&opt.Cluster.Hosts, "cluster.hosts", "", "集群内主机列表")

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

	buff, err := yaml.Marshal(opt)
	if err != nil {
		return "", fmt.Errorf("marshal config to yaml failed: %v", err)
	}
	opt.yamlStr = string(buff)

	if opt.ShowConfig {
		fmt.Printf("%s", opt.yamlStr)
	}

	return "", nil
}
