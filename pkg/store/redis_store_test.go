package store

import (
	"context"
	"testing"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/redis"

	"github.com/stretchr/testify/require"
)

func TestRedisStore_Id(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()
	require.Equal(t, "redis", s.Id())
}

func TestRedisStore_PutDeviceValidation(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()
	require.EqualError(t, s.PutDevice(nil), "device not be nil")
	require.EqualError(t, s.PutDevice(core.NewDevice("", "p", 0)), "device id must be present")
}

func TestRedisStore_PutDeviceRedisNil(t *testing.T) {
	// 无 client
	restore := redis.SetClientForTest(nil)
	t.Cleanup(restore)
	s := newTestRedisStore()
	err := s.PutDevice(core.NewDevice("d1", "p1", 0))
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis client not initialized")
}

func TestRedisStore_DeviceCRUDAndCache(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()

	require.Nil(t, s.GetDevice("dev-1"))

	dev := core.NewDevice("dev-1", "prod-a", 7)
	dev.Name = "meter"
	dev.DeviceType = core.DEVICE
	dev.ParentId = "gw-1"
	dev.ClusterId = "node-a"
	dev.Config = map[string]string{"k": "v", core.DEVICE_TIMEOUT_KEY: "60"}

	require.NoError(t, s.PutDevice(dev))

	// 缓存命中
	got := s.GetDevice("dev-1")
	require.NotNil(t, got)
	require.Equal(t, "dev-1", got.Id)
	require.Equal(t, "prod-a", got.ProductId)
	require.Equal(t, "meter", got.Name)
	require.Equal(t, core.DEVICE, got.DeviceType)
	require.Equal(t, "gw-1", got.ParentId)
	require.Equal(t, "node-a", got.ClusterId)
	require.Equal(t, int64(7), got.CreateId)
	require.Equal(t, "v", got.Config["k"])

	// 清内存缓存，强制走 Redis HGetAll
	syncMapClear(s)

	fromRedis := s.GetDevice("dev-1")
	require.NotNil(t, fromRedis)
	require.Equal(t, "dev-1", fromRedis.Id)
	require.Equal(t, "prod-a", fromRedis.ProductId)
	require.Equal(t, "meter", fromRedis.Name)
	require.Equal(t, "node-a", fromRedis.ClusterId)
	require.Equal(t, int64(7), fromRedis.CreateId)
	require.Equal(t, "v", fromRedis.Config["k"])

	s.DelDevice("dev-1")
	require.Nil(t, s.GetDevice("dev-1"))
}

// syncMapClear 清空内存缓存，迫使后续 Get 走 Redis。
func syncMapClear(s *redisDeviceStore) {
	s.cache.Range(func(k, _ any) bool {
		s.cache.Delete(k)
		return true
	})
}

func TestRedisStore_DeviceData(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()
	require.NoError(t, s.PutDevice(core.NewDevice("d1", "p", 0)))

	require.Equal(t, "", s.GetDeviceData("d1", "seq"))
	s.SetDeviceData("d1", "seq", "100")
	require.Equal(t, "100", s.GetDeviceData("d1", "seq"))
}

func TestRedisStore_ProductCRUDAndCache(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()

	require.Nil(t, s.GetProduct("p-1"))

	p, err := core.NewProduct("p-1", map[string]string{"x": "y"}, "es", `{"properties":[]}`)
	require.NoError(t, err)
	p.NetworkType = "MQTT_BROKER"
	require.NoError(t, s.PutProduct(p))

	got := s.GetProduct("p-1")
	require.NotNil(t, got)
	require.Equal(t, "p-1", got.Id)
	require.Equal(t, "es", got.StorePolicy)
	require.Equal(t, "MQTT_BROKER", got.NetworkType)
	require.Equal(t, "y", got.Config["x"])

	// 清缓存后从 Redis 回填
	syncMapClear(s)
	fromRedis := s.GetProduct("p-1")
	require.NotNil(t, fromRedis)
	require.Equal(t, "p-1", fromRedis.Id)
	require.Equal(t, "MQTT_BROKER", fromRedis.NetworkType)
	require.Equal(t, "y", fromRedis.Config["x"])

	s.DelProduct("p-1")
	// 删除后缓存与 Redis 均无
	syncMapClear(s)
	require.Nil(t, s.GetProduct("p-1"))
}


func TestRedisStore_PutProductValidation(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()
	require.EqualError(t, s.PutProduct(nil), "product not be nil")
	p, err := core.NewProduct("", nil, "", "")
	require.NoError(t, err)
	require.EqualError(t, s.PutProduct(p), "product id must be present")
}

func TestRedisStore_PutProductRedisNil(t *testing.T) {
	restore := redis.SetClientForTest(nil)
	t.Cleanup(restore)
	s := newTestRedisStore()
	p, err := core.NewProduct("p1", nil, "", "")
	require.NoError(t, err)
	err = s.PutProduct(p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis client not initialized")
}

func TestRedisStore_UpdateClusterId(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()
	dev := core.NewDevice("d1", "p", 0)
	require.NoError(t, s.PutDevice(dev))

	s.updateClusterId("d1")
	// 未配置集群时 localID 为空字符串，仍应写入并更新内存
	got := s.GetDevice("d1")
	require.NotNil(t, got)
	// ClusterId 与 cluster.GetClusterId() 一致（通常为空）
	ctx := context.Background()
	cid, err := redis.GetRedisClient().HGet(ctx, s.getDeviceKey("d1"), "clusterId").Result()
	require.NoError(t, err)
	require.Equal(t, cid, got.ClusterId)
}

func TestRedisStore_ClearClusterIdIfOwner(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()

	dev := core.NewDevice("d1", "p", 0)
	dev.ClusterId = "" // 本节点归属（空 local 时）
	require.NoError(t, s.PutDevice(dev))
	// 写入 zset 成员以便验证 ZRem
	ctx := context.Background()
	require.NoError(t, redis.GetRedisClient().ZAdd(ctx, s.getZKey(), &redis.Z{
		Score:  float64(time.Now().Unix() + 100),
		Member: "d1",
	}).Err())

	s.clearClusterIdIfOwner("d1")
	got := s.GetDevice("d1")
	require.NotNil(t, got)
	require.Equal(t, "", got.ClusterId)
	// zset 中应已移除
	n, err := redis.GetRedisClient().ZScore(ctx, s.getZKey(), "d1").Result()
	require.Error(t, err) // redis.Nil
	_ = n
}

func TestRedisStore_ClearClusterIdSkipsOtherOwner(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()
	dev := core.NewDevice("d1", "p", 0)
	dev.ClusterId = "other-node"
	require.NoError(t, s.PutDevice(dev))

	s.clearClusterIdIfOwner("d1")
	got := s.GetDevice("d1")
	require.Equal(t, "other-node", got.ClusterId)
}

func TestRedisStore_RefreshOfflineTimeout(t *testing.T) {
	mr := setupMiniRedis(t)
	s := newTestRedisStore()

	// 无设备：no-op
	s.RefreshOfflineTimeout("missing")

	dev := core.NewDevice("d1", "p", 0)
	// 无 timeout 配置：不入 zset
	require.NoError(t, s.PutDevice(dev))
	s.RefreshOfflineTimeout("d1")
	ctx := context.Background()
	members, err := redis.GetRedisClient().ZRange(ctx, s.getZKey(), 0, -1).Result()
	require.NoError(t, err)
	require.Empty(t, members)

	// 配置 offlineTimeout
	dev.Config[core.DEVICE_TIMEOUT_KEY] = "30"
	require.NoError(t, s.PutDevice(dev))
	s.RefreshOfflineTimeout("d1")
	members, err = redis.GetRedisClient().ZRange(ctx, s.getZKey(), 0, -1).Result()
	require.NoError(t, err)
	require.Equal(t, []string{"d1"}, members)

	score, err := redis.GetRedisClient().ZScore(ctx, s.getZKey(), "d1").Result()
	require.NoError(t, err)
	// score 约为 now+30
	now := float64(time.Now().Unix())
	require.InDelta(t, now+30, score, 3)
	require.True(t, mr.Exists(s.getZKey()))
}

func TestRedisStore_ScanOfflineDevicesRemovesExpired(t *testing.T) {
	setupMiniRedis(t)
	s := newTestRedisStore()
	dev := core.NewDevice("d-timeout", "p", 0)
	require.NoError(t, s.PutDevice(dev))

	ctx := context.Background()
	// 过期 score = 过去
	require.NoError(t, redis.GetRedisClient().ZAdd(ctx, s.getZKey(), &redis.Z{
		Score:  float64(time.Now().Unix() - 10),
		Member: "d-timeout",
	}).Err())
	// 未过期
	require.NoError(t, redis.GetRedisClient().ZAdd(ctx, s.getZKey(), &redis.Z{
		Score:  float64(time.Now().Unix() + 3600),
		Member: "d-future",
	}).Err())

	s.scanOfflineDevices()

	members, err := redis.GetRedisClient().ZRange(ctx, s.getZKey(), 0, -1).Result()
	require.NoError(t, err)
	require.Equal(t, []string{"d-future"}, members)
}
