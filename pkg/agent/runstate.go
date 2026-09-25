package agent

import (
	"context"
	"sync"
	"time"

	"go-iot/pkg/models"
	"go-iot/pkg/redis"
)

const (
	stuckGrace    = 45 * time.Second
	ConvKeyPrefix = "goiot:agent:conv:"
	CancelChannel = ConvKeyPrefix + "cancel"
	ConvSeqTTL    = 24 * time.Hour
)

// ConvRedisKey 返回会话维度的 Redis key，格式为 goiot:agent:conv:{convId}:{suffix}
func ConvRedisKey(convId, suffix string) string {
	if suffix == "" {
		return ConvKeyPrefix + convId
	}
	return ConvKeyPrefix + convId + ":" + suffix
}

type runSlot struct {
	cancel context.CancelFunc
	done   chan struct{}
}

var runSlots sync.Map // convId -> *runSlot

func (l *runLocks) Held(id string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.m[id]
	return ok
}

// IsLocked 检查当前会话是否已被加锁。优先检查本地内存锁，未命中时在 Redis 中探测锁 key。
func IsLocked(convId string) bool {
	if locks.Held(convId) {
		return true
	}
	if client := redis.GetRedisClient(); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		n, err := client.Exists(ctx, ConvRedisKey(convId, "lock")).Result()
		return err == nil && n > 0
	}
	return false
}

func registerRun(convId string, cancel context.CancelFunc) *runSlot {
	s := &runSlot{cancel: cancel, done: make(chan struct{})}
	runSlots.Store(convId, s)
	return s
}

func finishRun(convId string, s *runSlot) {
	if s != nil {
		select {
		case <-s.done:
		default:
			close(s.done)
		}
	}
	runSlots.Delete(convId)
}

// CancelActive cancels an in-process RunTurn. Returns false if this node is not running it.
func CancelActive(convId string) bool {
	v, ok := runSlots.Load(convId)
	if !ok {
		return false
	}
	s := v.(*runSlot)
	s.cancel()
	return true
}

func WaitRun(convId string, d time.Duration) {
	v, ok := runSlots.Load(convId)
	if !ok {
		return
	}
	s := v.(*runSlot)
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-s.done:
	case <-t.C:
	}
}

// RecoverStuck clears RunRunning/RunApplying left by a crashed process.
// If any cluster node still holds the lock, the run is live.
func RecoverStuck(store Store, conv *models.AgentConversation) {
	if conv == nil {
		return
	}
	if conv.RunStatus != RunRunning && conv.RunStatus != RunApplying {
		return
	}
	if IsLocked(conv.Id) {
		return
	}
	age := time.Since(time.Time(conv.UpdateTime))
	if age < stuckGrace {
		return
	}
	if pendingLeft(store, conv.Id) {
		conv.RunStatus = RunAwaitingConfirm
	} else {
		conv.RunStatus = RunIdle
	}
	conv.NeedsResume = false
	_ = store.SaveConversation(conv)
}

func SyncRunStatus(store Store, conv *models.AgentConversation) {
	RecoverStuck(store, conv)
	if conv.RunStatus == RunAwaitingConfirm && !pendingLeft(store, conv.Id) {
		conv.RunStatus = RunIdle
		_ = store.SaveConversation(conv)
	}
}

var cancelSubOnce sync.Once

func resetCancelSubscriberForTest() {
	cancelSubOnce = sync.Once{}
}

// StartCancelSubscriber 启动 Redis 跨节点取消广播监听。
func StartCancelSubscriber() {
	cancelSubOnce.Do(func() {
		client := redis.GetRedisClient()
		if client == nil {
			return
		}
		go func() {
			for msg := range redis.Sub(CancelChannel) {
				if msg != nil && msg.Payload != "" {
					CancelActive(msg.Payload)
				}
			}
		}()
	})
}

// BroadcastCancel 向全集群广播取消信号，使执行该会话的节点停止运行。
func BroadcastCancel(convId string) {
	StartCancelSubscriber()
	if client := redis.GetRedisClient(); client != nil {
		redis.Pub(CancelChannel, convId)
	}
}
