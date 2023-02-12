package redis

import (
	"context"
	"fmt"
	"go-iot/pkg/core/boot"
	"strconv"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/go-redis/redis/v8"
)

func init() {
	boot.AddStartLinstener(func() {
		GetRedisClient()
	})
}

// the config of redis
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

func (r RedisConfig) String() string {
	return fmt.Sprintf("[addr=%s, db=%v, poolSize=%v]", r.Addr, r.DB, r.PoolSize)
}

var DefaultRedisConfig RedisConfig = RedisConfig{
	Addr:     "127.0.0:6379",
	PoolSize: 10,
}

func Config(fn func(key string, call func(string))) {
	fn("redis.addr", func(s string) {
		DefaultRedisConfig.Addr = s
	})
	fn("redis.password", func(s string) {
		DefaultRedisConfig.Password = s
	})
	fn("redis.db", func(s string) {
		num, err := strconv.Atoi(s)
		if err == nil {
			DefaultRedisConfig.DB = num
		} else {
			logs.Error("redis.db error:", err)
		}
	})
	logs.Info("redis config: ", DefaultRedisConfig)
}

var rdb *redis.Client

func GetRedisClient() *redis.Client {
	if rdb == nil {
		var mutex sync.Mutex
		mutex.Lock()
		defer mutex.Unlock()
		rdb = redis.NewClient(&redis.Options{
			Addr:     DefaultRedisConfig.Addr,
			Password: DefaultRedisConfig.Password,
			DB:       DefaultRedisConfig.DB,
			PoolSize: DefaultRedisConfig.PoolSize,
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err := rdb.Ping(ctx).Err()
		if err != nil {
			panic(fmt.Sprintf("redis connect error: %v", DefaultRedisConfig))
		}
	}
	return rdb
}
