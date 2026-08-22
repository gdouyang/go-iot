// redis相关配置与方法
package redis

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	logs "go-iot/pkg/logger"
	"go-iot/pkg/option"

	"github.com/go-redis/redis/v8"
)

// the config of redis
type RedisConfig struct {
	Addr     string
	Addrs    []string
	Password string
	DB       int
	PoolSize int
}

type ZRangeBy = redis.ZRangeBy
type Z = redis.Z

// Script/NewScript 透传 go-redis 脚本支持（Lua 原子操作用）
type Script = redis.Script

var NewScript = redis.NewScript

func (r RedisConfig) String() string {
	if len(r.Addrs) > 1 {
		return fmt.Sprintf("[cluster addrs=%v, poolSize=%v]", r.Addrs, r.PoolSize)
	}
	return fmt.Sprintf("[addr=%s, db=%v, poolSize=%v]", r.Addr, r.DB, r.PoolSize)
}

var DefaultRedisConfig RedisConfig = RedisConfig{
	Addr:     "127.0.0:6379",
	PoolSize: 100,
}

// config redis
func Config(opt *option.Options) {
	DefaultRedisConfig.Addr = opt.Redis.Addr
	// addr 逗号分隔多地址时自动启用 Cluster 模式（多节点）；单地址保持单实例
	DefaultRedisConfig.Addrs = splitAddr(opt.Redis.Addr)
	DefaultRedisConfig.Password = opt.Redis.Password
	DefaultRedisConfig.DB = opt.Redis.Db
	// 仅在配置了有效值时覆盖默认连接池大小；≤0 保持默认(100)，
	// 大并发设备上线场景每个上线都要查离线命令队列，池子太小会 connection pool timeout
	if opt.Redis.PoolSize > 0 {
		DefaultRedisConfig.PoolSize = opt.Redis.PoolSize
	}
	// Cluster 模式不支持 select db，配置了 db 时提示
	if len(DefaultRedisConfig.Addrs) > 1 && opt.Redis.Db != 0 {
		logs.Warnf("redis cluster mode does not support db selection, db=%d ignored", opt.Redis.Db)
	}
	logs.Infof("redis config: %v", DefaultRedisConfig)
	InitRedis()
}

// splitAddr 逗号分隔多节点地址（addr: "a:6379,b:6379" → 集群模式）
func splitAddr(addr string) []string {
	parts := make([]string, 0)
	for _, p := range strings.Split(addr, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

var rdb redis.UniversalClient

const Nil = redis.Nil

// init redis client, panic can't connect to the server
func InitRedis() {
	if rdb == nil {
		var mutex sync.Mutex
		mutex.Lock()
		defer mutex.Unlock()
		for {
			if len(DefaultRedisConfig.Addrs) > 1 {
				// addr 逗号分隔多节点 → Redis Cluster 模式（水平扩展/高可用）
				rdb = redis.NewClusterClient(&redis.ClusterOptions{
					Addrs:    DefaultRedisConfig.Addrs,
					Password: DefaultRedisConfig.Password,
					PoolSize: DefaultRedisConfig.PoolSize,
				})
			} else {
				// 单实例模式（默认）：单地址或未配置
				addr := DefaultRedisConfig.Addr
				if len(DefaultRedisConfig.Addrs) == 1 {
					addr = DefaultRedisConfig.Addrs[0]
				}
				rdb = redis.NewClient(&redis.Options{
					Addr:     addr,
					Password: DefaultRedisConfig.Password,
					DB:       DefaultRedisConfig.DB,
					PoolSize: DefaultRedisConfig.PoolSize,
				})
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			err := rdb.Ping(ctx).Err()
			cancel()
			if err == nil {
				break // 连接成功，退出循环
			}
			logs.Errorf("redis error: %v", err)
			time.Sleep(5 * time.Second) // 等待5秒后重试
		}
	}
}

func GetRedisClient() redis.UniversalClient {
	return rdb
}

// SetClientForTest 注入 Redis 客户端（单测用 miniredis 等）。返回还原函数。
// 生产代码请使用 Config / InitRedis，勿调用本函数。
func SetClientForTest(c redis.UniversalClient) (restore func()) {
	old := rdb
	rdb = c
	return func() { rdb = old }
}

func Sub(channels ...string) <-chan *redis.Message {
	client := GetRedisClient()
	sub := client.Subscribe(client.Context(), channels...)
	return sub.Channel()
}

func Pub(channel string, message interface{}) {
	client := GetRedisClient()
	client.Publish(client.Context(), channel, message).Result()
}
