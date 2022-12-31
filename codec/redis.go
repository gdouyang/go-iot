package codec

import (
	"context"
	"fmt"
	"sync"
	"time"

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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err := rdb.Ping(ctx).Err()
		if err != nil {
			panic(fmt.Sprintf("redis connect error: %v", DefaultRedisConfig))
		}
	}
	return rdb
}
