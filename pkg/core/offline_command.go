package core

import (
	"encoding/json"
	"go-iot/pkg/common"
	"sync"
)

// OfflineCommandQueue 设备离线时功能调用命令队列。
// 实现放在 infra（store/redis），core 只依赖接口，避免 domain → redis。
type OfflineCommandQueue interface {
	// Enqueue 缓存一条离线命令；成功时返回 Code=200 的 *common.Err（与历史语义一致）。
	Enqueue(message FuncInvoke) *common.Err
	// TakeAll 取出并清空该设备的全部缓存命令。
	TakeAll(deviceId string) []FuncInvoke
}

// MemoryOfflineCommandQueue 内存实现，单测与无 Redis 场景。
type MemoryOfflineCommandQueue struct {
	mu sync.Mutex // 离线命令下发（API 并发）与设备上线 TakeAll 并发访问 data
	// deviceId -> JSON payloads (same shape as redis list)
	data map[string][]string
	max  int
}

func NewMemoryOfflineCommandQueue() *MemoryOfflineCommandQueue {
	return &MemoryOfflineCommandQueue{
		data: make(map[string][]string),
		max:  20,
	}
}

func (q *MemoryOfflineCommandQueue) Enqueue(message FuncInvoke) *common.Err {
	if message.FunctionId == OTA_UPDATE {
		return common.NewErr400("设备离线，不支持OTA升级")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	list := q.data[message.DeviceId]
	if len(list) >= q.max {
		return common.NewErr400("设备离线，命令缓存队列已满，请稍后再试")
	}
	b, _ := json.Marshal(message)
	q.data[message.DeviceId] = append(list, string(b))
	return common.NewErr(200, "设备离线，命令已缓存")
}

func (q *MemoryOfflineCommandQueue) TakeAll(deviceId string) []FuncInvoke {
	q.mu.Lock()
	defer q.mu.Unlock()
	list := q.data[deviceId]
	delete(q.data, deviceId)
	out := make([]FuncInvoke, 0, len(list))
	for _, s := range list {
		var message FuncInvoke
		if err := json.Unmarshal([]byte(s), &message); err == nil {
			out = append(out, message)
		}
	}
	return out
}

// NoopOfflineCommandQueue 不缓存；Enqueue 返回设备离线错误。
type NoopOfflineCommandQueue struct{}

func (NoopOfflineCommandQueue) Enqueue(message FuncInvoke) *common.Err {
	return common.NewErr400("设备已离线")
}

func (NoopOfflineCommandQueue) TakeAll(deviceId string) []FuncInvoke {
	return nil
}

var defaultOfflineQueue OfflineCommandQueue = NewMemoryOfflineCommandQueue()

// RegOfflineCommandQueue 注册离线命令队列。
func RegOfflineCommandQueue(q OfflineCommandQueue) {
	if q == nil {
		return
	}
	defaultOfflineQueue = q
}

// GetOfflineCommandQueue 返回当前离线命令队列。
func GetOfflineCommandQueue() OfflineCommandQueue {
	return defaultOfflineQueue
}
