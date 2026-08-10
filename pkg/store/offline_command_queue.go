package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-iot/pkg/common"
	"go-iot/pkg/core"
	"go-iot/pkg/redis"

	logs "go-iot/pkg/logger"
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
	b, err := json.Marshal(message)
	if err != nil {
		logs.Errorf("offline queue marshal error: %v", err)
		return common.NewErr500("离线命令序列化失败")
	}
	ctx := context.Background()
	n, err := client.LLen(ctx, key).Result()
	if err != nil {
		logs.Errorf("offline queue LLen error: %v", err)
		return common.NewErr500("redis 查询离线命令队列失败")
	}
	if n >= int64(q.maxLen) {
		return common.NewErr400("设备离线，命令缓存队列已满，请稍后再试")
	}
	if err := client.RPush(ctx, key, string(b)).Err(); err != nil {
		logs.Errorf("offline queue RPush error: %v", err)
		return common.NewErr500("redis 缓存离线命令失败")
	}
	if err := client.Expire(ctx, key, q.ttl).Err(); err != nil {
		logs.Errorf("offline queue Expire error: %v", err)
	}
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
	if err != nil {
		logs.Errorf("offline queue LRange error: %v", err)
		return nil
	}
	if len(cmds) == 0 {
		return nil
	}
	if err := client.Del(ctx, key).Err(); err != nil {
		logs.Errorf("offline queue Del error: %v", err)
	}

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
