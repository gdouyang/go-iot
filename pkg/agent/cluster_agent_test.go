package agent

import (
	"context"
	"testing"
	"time"

	"go-iot/pkg/models"
	"go-iot/pkg/redis"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) *miniredis.Miniredis {
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

func TestAgent_RedisLockMutualExclusion(t *testing.T) {
	setupTestRedis(t)
	convId := "test-conv-" + NewHexID()

	// 节点 A 加锁成功
	require.True(t, TryLock(convId))

	// 节点 B 尝试加同一会话的锁，应当被拒绝
	require.False(t, TryLock(convId))

	// 检查 IsLocked 状态为 true
	require.True(t, IsLocked(convId))

	// 节点 A 解锁
	Unlock(convId)

	// 解锁后节点 B 可以成功加锁
	require.True(t, TryLock(convId))
	Unlock(convId)
}

func TestAgent_RedisSeqNoIncrement(t *testing.T) {
	mr := setupTestRedis(t)
	convId := "test-conv-seq-" + NewHexID()

	s1 := NextSeqNo(convId)
	s2 := NextSeqNo(convId)
	s3 := NextSeqNo(convId)

	require.Equal(t, int64(1), s1)
	require.Equal(t, int64(2), s2)
	require.Equal(t, int64(3), s3)

	m := &models.AgentMessage{Id: NewHexID(), ConversationId: convId}
	StampMessage(m)
	require.Equal(t, int64(4), m.SeqNo)

	// 验证 key 的 TTL 已设置为 24 小时
	ttl := mr.TTL(ConvRedisKey(convId, "seq"))
	require.True(t, ttl > 0 && ttl <= 24*time.Hour)
}

func TestAgent_EnsureConvSeq(t *testing.T) {
	mr := setupTestRedis(t)
	convId := "test-conv-ensure-" + NewHexID()
	key := ConvRedisKey(convId, "seq")

	store := NewMemoryStore()
	// 模拟已存在最大 SeqNo 为 5 的消息
	_ = store.SaveMessage(&models.AgentMessage{Id: NewHexID(), ConversationId: convId, SeqNo: 2})
	_ = store.SaveMessage(&models.AgentMessage{Id: NewHexID(), ConversationId: convId, SeqNo: 5})

	// 此时 Redis 中尚无此 key
	require.False(t, mr.Exists(key))

	// 开始对话时调用 EnsureConvSeq
	EnsureConvSeq(context.Background(), convId, store)

	// key 应当被创建，初始值为 5
	require.True(t, mr.Exists(key))
	val, err := mr.Get(key)
	require.NoError(t, err)
	require.Equal(t, "5", val)
	require.True(t, mr.TTL(key) > 0 && mr.TTL(key) <= 24*time.Hour)

	// 后续新消息自增，得到 6
	next := NextSeqNo(convId)
	require.Equal(t, int64(6), next)

	// 再次调用 EnsureConvSeq，已有 key 不会被重置
	EnsureConvSeq(context.Background(), convId, store)
	next2 := NextSeqNo(convId)
	require.Equal(t, int64(7), next2)
}

func TestAgent_BroadcastCancel(t *testing.T) {
	setupTestRedis(t)
	resetCancelSubscriberForTest()
	StartCancelSubscriber()
	time.Sleep(50 * time.Millisecond)

	convId := "test-conv-cancel-" + NewHexID()
	canceled := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	// 注册本地运行槽位
	go func() {
		<-ctx.Done()
		close(canceled)
	}()

	// 模拟当前节点注册了该会话
	slot := registerRun(convId, cancel)
	defer finishRun(convId, slot)

	// 广播取消（模拟其他节点发出取消请求）
	BroadcastCancel(convId)

	// 校验本地 context 被成功取消
	select {
	case <-canceled:
		// 成功收到取消信号
	case <-time.After(4 * time.Second):
		t.Fatal("timeout waiting for cancel broadcast")
	}
}
