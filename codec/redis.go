package codec

import (
	"sync"

	"github.com/go-redis/redis/v8"
)

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
	}
	return rdb
}
