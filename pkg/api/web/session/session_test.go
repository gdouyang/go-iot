package session

import (
	"context"
	"strconv"
	"testing"
	"time"

	"go-iot/pkg/redis"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestRandomSessionIDUniqueness(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id, err := randomSessionID()
		require.NoError(t, err)
		require.Len(t, id, sessionIDBytes*2) // hex
		_, dup := seen[id]
		require.False(t, dup, "collision at %d", i)
		seen[id] = struct{}{}
	}
}

func TestRandomSessionIDNotTrivial(t *testing.T) {
	a, err := randomSessionID()
	require.NoError(t, err)
	b, err := randomSessionID()
	require.NoError(t, err)
	require.NotEqual(t, a, b)
	require.Len(t, a, 64)
	require.Regexp(t, `^[0-9a-f]{64}$`, a)
}

func TestTTLDefaults(t *testing.T) {
	require.Equal(t, DefaultExpireSec, (*HttpSession)(nil).TTL())
	require.Equal(t, DefaultExpireSec, (&HttpSession{}).TTL())
	require.Equal(t, DefaultExpireSec, (&HttpSession{ExpireSec: 0}).TTL())
	require.Equal(t, 7200, (&HttpSession{ExpireSec: 7200}).TTL())
}

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

func TestNewSessionPersistsExpireAndTTL(t *testing.T) {
	mr := setupMiniRedis(t)

	s := NewSession(120)
	require.Len(t, s.Sessionid, 64)
	require.Equal(t, 120, s.ExpireSec)

	key := getSessionId(s.Sessionid)
	// expire 字段已 HSET
	raw, err := redis.GetRedisClient().HGet(context.Background(), key, "expire").Result()
	require.NoError(t, err)
	require.Equal(t, "120", raw)

	// Redis key TTL 约为 120s
	ttl := mr.TTL(key)
	require.Greater(t, ttl, 100*time.Second)
	require.LessOrEqual(t, ttl, 120*time.Second)
}

func TestNewSessionDefaultExpire(t *testing.T) {
	mr := setupMiniRedis(t)
	s := NewSession(0)
	require.Equal(t, DefaultExpireSec, s.ExpireSec)
	ttl := mr.TTL(getSessionId(s.Sessionid))
	require.Greater(t, ttl, time.Duration(DefaultExpireSec-10)*time.Second)
	require.LessOrEqual(t, ttl, time.Duration(DefaultExpireSec)*time.Second)
}

func TestGetSlidesTTLAndReadsExpire(t *testing.T) {
	mr := setupMiniRedis(t)

	s := NewSession(90)
	key := getSessionId(s.Sessionid)

	// 人为缩短剩余 TTL，模拟快过期
	mr.SetTTL(key, 5*time.Second)
	require.LessOrEqual(t, mr.TTL(key), 5*time.Second)

	// Get 应滑动续期回 ~90s
	got := Get(s.Sessionid)
	require.NotNil(t, got)
	require.Equal(t, s.Sessionid, got.Sessionid)
	require.Equal(t, 90, got.ExpireSec)

	ttl := mr.TTL(key)
	require.Greater(t, ttl, 80*time.Second)
	require.LessOrEqual(t, ttl, 90*time.Second)
}

func TestGetMissingSession(t *testing.T) {
	setupMiniRedis(t)
	require.Nil(t, Get(""))
	require.Nil(t, Get("not-exist-session-id"))
}

func TestGetWithoutStoredExpireUsesDefault(t *testing.T) {
	mr := setupMiniRedis(t)
	// 模拟旧数据：只有 hash 无 expire 字段
	id := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	key := getSessionId(id)
	client := redis.GetRedisClient()
	require.NoError(t, client.HSet(context.Background(), key, "user", `{}`).Err())
	require.NoError(t, client.Expire(context.Background(), key, 10*time.Second).Err())

	got := Get(id)
	require.NotNil(t, got)
	require.Equal(t, DefaultExpireSec, got.ExpireSec)
	ttl := mr.TTL(key)
	require.Greater(t, ttl, time.Duration(DefaultExpireSec-10)*time.Second)
}

func TestUpdateExpireUsesTTL(t *testing.T) {
	mr := setupMiniRedis(t)
	s := &HttpSession{Sessionid: "11223344556677889900aabbccddeeff11223344556677889900aabbccddeeff", ExpireSec: 60}
	client := redis.GetRedisClient()
	key := s.getSessionId()
	require.NoError(t, client.HSet(context.Background(), key, "expire", strconv.Itoa(60)).Err())

	s.UpdateExpire()
	ttl := mr.TTL(key)
	require.Greater(t, ttl, 50*time.Second)
	require.LessOrEqual(t, ttl, 60*time.Second)
}
