package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-iot/pkg/common"
	"go-iot/pkg/core"
	"go-iot/pkg/redis"
)

// RedisOfflineCommandQueue 基于 Redis List 的离线命令队列（与历史 key/TTL 兼容）。
type RedisOfflineCommandQueue struct {
	maxLen int
	ttl    time.Duration
}

// NewRedisOfflineCommandQueue 使用默认上限 20 条、TTL 48h。
func NewRedisOfflineCommandQueue() *RedisOfflineCommandQueue {
	return &RedisOfflineCommandQueue{
		maxLen: 20,
		ttl:    time.Hour * 48,
	}
}

func offlineRedisKey(deviceId string) string {
	return fmt.Sprintf("goiot:device_offline_cmd:%s", deviceId)
}

func (q *RedisOfflineCommandQueue) Enqueue(message core.FuncInvoke) *common.Err {
	if message.FunctionId == core.OTA_UPDATE {
		return common.NewErr400("设备离线，不支持OTA升级")
	}
	client := redis.GetRedisClient()
	if client == nil {
		return common.NewErr500("redis not available")
	}
	key := offlineRedisKey(message.DeviceId)
	b, _ := json.Marshal(message)
	ctx := context.Background()
	if client.LLen(ctx, key).Val() >= int64(q.maxLen) {
		return common.NewErr400("设备离线，命令缓存队列已满，请稍后再试")
	}
	client.RPush(ctx, key, string(b))
	client.Expire(ctx, key, q.ttl)
	return common.NewErr(200, "设备离线，命令已缓存")
}

func (q *RedisOfflineCommandQueue) TakeAll(deviceId string) []core.FuncInvoke {
	client := redis.GetRedisClient()
	if client == nil {
		return nil
	}
	key := offlineRedisKey(deviceId)
	ctx := context.Background()
	cmds, err := client.LRange(ctx, key, 0, -1).Result()
	if err != nil || len(cmds) == 0 {
		return nil
	}
	client.Del(ctx, key)

	out := make([]core.FuncInvoke, 0, len(cmds))
	for _, cmdStr := range cmds {
		var message core.FuncInvoke
		if err := json.Unmarshal([]byte(cmdStr), &message); err == nil {
			out = append(out, message)
		}
	}
	return out
}

var _ core.OfflineCommandQueue = (*RedisOfflineCommandQueue)(nil)
