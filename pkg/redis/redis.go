// redis相关配置与方法
package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	logs "go-iot/pkg/logger"
	"go-iot/pkg/option"

	"github.com/go-redis/redis/v8"
)

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

// config redis
func Config(opt *option.Options) {
	DefaultRedisConfig.Addr = opt.Redis.Addr
	DefaultRedisConfig.Password = opt.Redis.Password
	DefaultRedisConfig.DB = opt.Redis.Db
	logs.Infof("redis config: %v", DefaultRedisConfig)
	InitRedis()
}

var rdb *redis.Client

const Nil = redis.Nil

// init redis client, panic can't connect to the server
func InitRedis() {
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
			logs.Errorf(err.Error())
			panic(fmt.Sprintf("redis connect error: %v", err))
		}
	}
}

func GetRedisClient() *redis.Client {
	return rdb
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
