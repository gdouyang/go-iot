package store

import (
	"context"
	"testing"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/redis"

	"github.com/stretchr/testify/require"
)

func TestOfflineQueue_EnqueueOTARejected(t *testing.T) {
	setupMiniRedis(t)
	q := NewRedisOfflineCommandQueue()
	err := q.Enqueue(core.FuncInvoke{DeviceId: "d1", FunctionId: core.OTA_UPDATE})
	require.NotNil(t, err)
	require.Equal(t, 400, err.Code)
	require.Contains(t, err.Message, "OTA")
}

func TestOfflineQueue_RedisNil(t *testing.T) {
	restore := redis.SetClientForTest(nil)
	t.Cleanup(restore)
	q := NewRedisOfflineCommandQueue()
	err := q.Enqueue(core.FuncInvoke{DeviceId: "d1", FunctionId: "fn"})
	require.NotNil(t, err)
	require.Equal(t, 500, err.Code)
	require.Nil(t, q.TakeAll("d1"))
}

func TestOfflineQueue_EnqueueAndTakeAll(t *testing.T) {
	mr := setupMiniRedis(t)
	q := NewRedisOfflineCommandQueue()

	msg1 := core.FuncInvoke{
		DeviceId:   "dev-a",
		FunctionId: "switch",
		Data:       map[string]interface{}{"on": true},
		TraceId:    "t1",
	}
	msg2 := core.FuncInvoke{DeviceId: "dev-a", FunctionId: "reset", TraceId: "t2"}

	e1 := q.Enqueue(msg1)
	require.NotNil(t, e1)
	require.Equal(t, 200, e1.Code)
	require.Contains(t, e1.Message, "已缓存")

	e2 := q.Enqueue(msg2)
	require.Equal(t, 200, e2.Code)

	key := offlineRedisKey("dev-a")
	require.True(t, mr.Exists(key))
	// TTL 约 48h
	ttl := mr.TTL(key)
	require.Greater(t, ttl, 47*time.Hour)

	all := q.TakeAll("dev-a")
	require.Len(t, all, 2)
	require.Equal(t, "switch", all[0].FunctionId)
	require.Equal(t, "t1", all[0].TraceId)
	require.Equal(t, true, all[0].Data["on"])
	require.Equal(t, "reset", all[1].FunctionId)

	// TakeAll 后删除队列
	require.False(t, mr.Exists(key))
	require.Empty(t, q.TakeAll("dev-a"))
}

func TestOfflineQueue_MaxLen(t *testing.T) {
	setupMiniRedis(t)
	q := &RedisOfflineCommandQueue{maxLen: 3, ttl: time.Hour}

	for i := 0; i < 3; i++ {
		err := q.Enqueue(core.FuncInvoke{DeviceId: "d1", FunctionId: "f"})
		require.Equal(t, 200, err.Code, "i=%d", i)
	}
	err := q.Enqueue(core.FuncInvoke{DeviceId: "d1", FunctionId: "overflow"})
	require.Equal(t, 400, err.Code)
	require.Contains(t, err.Message, "已满")

	all := q.TakeAll("d1")
	require.Len(t, all, 3)
}

func TestOfflineQueue_TakeAllEmpty(t *testing.T) {
	setupMiniRedis(t)
	q := NewRedisOfflineCommandQueue()
	require.Empty(t, q.TakeAll("no-device"))
}

func TestOfflineQueue_TakeAllSkipsBadJSON(t *testing.T) {
	setupMiniRedis(t)
	q := NewRedisOfflineCommandQueue()
	ctx := context.Background()
	key := offlineRedisKey("d1")
	client := redis.GetRedisClient()
	require.NoError(t, client.RPush(ctx, key, `{"functionId":"ok","deviceId":"d1"}`, `not-json`, `{"functionId":"ok2","deviceId":"d1"}`).Err())

	all := q.TakeAll("d1")
	require.Len(t, all, 2)
	require.Equal(t, "ok", all[0].FunctionId)
	require.Equal(t, "ok2", all[1].FunctionId)
}

func TestOfflineRedisKey(t *testing.T) {
	require.Equal(t, "goiot:device_offline_cmd:abc", offlineRedisKey("abc"))
}
