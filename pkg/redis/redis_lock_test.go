package redis_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-iot/pkg/redis"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, goredis.UniversalClient) {
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
	return mr, client
}

func TestLock_TryLockAndUnlock(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	key := "test:lock:try"

	lock1 := redis.NewLock(key, redis.WithTTL(5*time.Second), redis.WithClient(client))
	ok, err := lock1.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, ok)

	// 第二个客户端尝试加同一把锁，应当获取失败
	lock2 := redis.NewLock(key, redis.WithTTL(5*time.Second), redis.WithClient(client))
	ok, err = lock2.TryLock(ctx)
	require.NoError(t, err)
	require.False(t, ok)

	// lock1 释放锁
	err = lock1.Unlock(ctx)
	require.NoError(t, err)

	// lock2 再次尝试，应当成功
	ok, err = lock2.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, lock2.Unlock(ctx))
}

func TestLock_UnlockNotHeld(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	key := "test:lock:not_held"

	lock1 := redis.NewLock(key, redis.WithTTL(5*time.Second), redis.WithClient(client))
	// 锁尚未获取时执行 unlock
	err := lock1.Unlock(ctx)
	require.Equal(t, redis.ErrLockNotHeld, err)

	// lock1 获取锁
	ok, err := lock1.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, ok)

	// lock2 尝试释放 lock1 的锁，应当返回 ErrLockNotHeld
	lock2 := redis.NewLock(key, redis.WithTTL(5*time.Second), redis.WithClient(client))
	err = lock2.Unlock(ctx)
	require.Equal(t, redis.ErrLockNotHeld, err)

	// lock1 正常释放
	require.NoError(t, lock1.Unlock(ctx))
}

func TestLock_Refresh(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	key := "test:lock:refresh"

	lock := redis.NewLock(key, redis.WithTTL(3*time.Second), redis.WithClient(client))
	ok, err := lock.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, ok)

	// 前进 2 秒
	mr.FastForward(2 * time.Second)
	ttl := mr.TTL(key)
	require.True(t, ttl <= 1*time.Second && ttl > 0)

	// 续期
	err = lock.Refresh(ctx)
	require.NoError(t, err)

	// 验证 TTL 恢复为 3 秒
	ttlAfter := mr.TTL(key)
	require.True(t, ttlAfter > 2*time.Second)

	require.NoError(t, lock.Unlock(ctx))
}

func TestLock_AutoRefreshWatchdog(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	key := "test:lock:watchdog"

	// 使用自动看门狗续期
	lock := redis.NewLock(key, redis.WithTTL(600*time.Millisecond), redis.WithAutoRefresh(true), redis.WithClient(client))
	ok, err := lock.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, ok)

	// 等待超过 1 次续约周期，锁依然有效
	time.Sleep(500 * time.Millisecond)

	// 此时别的客户端应当仍然拿不到锁
	other := redis.NewLock(key, redis.WithTTL(time.Second), redis.WithClient(client))
	ok, err = other.TryLock(ctx)
	require.NoError(t, err)
	require.False(t, ok)

	// 解锁，看门狗停止
	require.NoError(t, lock.Unlock(ctx))

	// 解锁后其他客户端立即可获取
	ok, err = other.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, other.Unlock(ctx))
}

func TestLock_AcquireLockBlocking(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	key := "test:lock:blocking"

	// 先被 lock1 持有
	lock1, err := redis.AcquireLock(ctx, key, 2*time.Second, redis.WithClient(client))
	require.NoError(t, err)
	require.NotNil(t, lock1)

	var lock2Acquired int32
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		// lock2 阻塞获取锁
		lock2, err := redis.AcquireLock(ctx, key, 2*time.Second, redis.WithRetryInterval(20*time.Millisecond), redis.WithClient(client))
		if err == nil && lock2 != nil {
			atomic.StoreInt32(&lock2Acquired, 1)
			_ = lock2.Unlock(ctx)
		}
	}()

	// 保证 goroutine 已经开始轮询
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, int32(0), atomic.LoadInt32(&lock2Acquired))

	// lock1 释放锁
	require.NoError(t, lock1.Unlock(ctx))

	// 等待 lock2 成功获取
	wg.Wait()
	require.Equal(t, int32(1), atomic.LoadInt32(&lock2Acquired))
}

func TestLock_AcquireLockTimeout(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	key := "test:lock:timeout"

	lock1, ok, err := redis.TryAcquireLock(ctx, key, 5*time.Second, redis.WithClient(client))
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, lock1)

	// 用带短超时的 context 尝试阻塞获取
	timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	lock2, err := redis.AcquireLock(timeoutCtx, key, 5*time.Second, redis.WithRetryInterval(20*time.Millisecond), redis.WithClient(client))
	require.Error(t, err)
	require.Nil(t, lock2)
	require.Equal(t, context.DeadlineExceeded, err)

	require.NoError(t, lock1.Unlock(ctx))
}
