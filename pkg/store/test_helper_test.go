package store

import (
	"sync"
	"testing"

	"go-iot/pkg/redis"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

// setupMiniRedis 注入 miniredis，测试结束自动还原。
func setupMiniRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	restore := redis.SetClientForTest(client)
	t.Cleanup(func() {
		_ = client.Close()
		restore()
	})
	return mr
}

// newTestRedisStore 不订阅 eventbus、不启 offline scanner，便于单测 CRUD。
func newTestRedisStore() *redisDeviceStore {
	return &redisDeviceStore{cache: sync.Map{}}
}
