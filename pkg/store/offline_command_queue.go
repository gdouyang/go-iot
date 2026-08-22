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

// takeAllScript 原子取出并清空设备离线命令队列：
// LRange 与 Del 必须在 Redis 服务端单次执行（Lua），否则两步之间存在窗口，
// 设备快速断开重连时两个并发 TakeAll 会读到同一批命令导致重复下发
var takeAllScript = redis.NewScript(`
local cmds = redis.call('LRANGE', KEYS[1], 0, -1)
redis.call('DEL', KEYS[1])
return cmds
`)

func (q *RedisOfflineCommandQueue) TakeAll(deviceId string) []core.FuncInvoke {
	client := redis.GetRedisClient()
	if client == nil {
		return nil
	}
	ctx := context.Background()
	cmds, err := takeAllScript.Run(ctx, client, []string{offlineRedisKey(deviceId)}).Slice()
	if err != nil {
		// 队列为空时脚本返回空 table，Slice() 会报 redis.Nil，属正常情况
		if err != redis.Nil {
			logs.Errorf("offline queue TakeAll error: %v", err)
		}
		return nil
	}

	out := make([]core.FuncInvoke, 0, len(cmds))
	for _, cmdStr := range cmds {
		if s, ok := cmdStr.(string); ok {
			var message core.FuncInvoke
			if err := json.Unmarshal([]byte(s), &message); err == nil {
				out = append(out, message)
			}
		}
	}
	return out
}

var _ core.OfflineCommandQueue = (*RedisOfflineCommandQueue)(nil)
