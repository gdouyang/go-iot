package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/redis"
	"go-iot/pkg/util"
	"strconv"
	"sync"
	"time"

	logs "go-iot/pkg/logger"
)

func NewRedisStore() core.DeviceStore {
	var redisStore = &redisDeviceStore{cache: sync.Map{}}
	redisStore.init()
	return redisStore
}

// device store for redis
type redisDeviceStore struct {
	cache sync.Map
}

func (store *redisDeviceStore) Id() string {
	return "redis"
}

func (store *redisDeviceStore) init() {
	eventbus.Subscribe("/device/*/*/*", func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OnlineMessage); ok {
			store.updateClusterId(m.DeviceId)
			store.RefreshOfflineTimeout(m.DeviceId)
		}
		if m, ok := msg.(*eventbus.OfflineMessage); ok {
			// 仅当本节点仍是归属时清空，避免重连到其它节点后被旧 offline 清掉
			store.clearClusterIdIfOwner(m.DeviceId)
		}
		if m, ok := msg.(*eventbus.PropertiesMessage); ok {
			store.RefreshOfflineTimeout(m.DeviceId)
		}
	})
	go store.startOfflineScanner()
}

func (store *redisDeviceStore) getDeviceKey(deviceId string) string {
	return "goiot:device:" + deviceId
}

func (store *redisDeviceStore) getDevice(deviceId string) (*core.Device, bool) {
	device, ok := store.cache.Load(store.getDeviceKey(deviceId))
	if ok {
		if device != nil {
			return device.(*core.Device), true
		}
		return nil, true
	}
	return nil, false
}

func (m *redisDeviceStore) GetDevice(deviceId string) *core.Device {
	device, ok := m.getDevice(deviceId)
	if ok {
		return device
	}
	if device == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		rdb := redis.GetRedisClient()
		deviceKey := m.getDeviceKey(deviceId)
		data, err := rdb.HGetAll(ctx, deviceKey).Result()
		if err != nil {
			if err != redis.Nil {
				logs.Errorf("hgetall device error: %v", err)
			}
			return nil
		}
		device = core.NewDevice(data["id"], data["productId"], 0)
		device.Name = data["name"]
		device.DeviceType = data["devType"]
		device.ParentId = data["parentId"]
		device.ClusterId = data["clusterId"]
		if str, ok := data["config"]; ok {
			err = json.Unmarshal([]byte(str), &device.Config)
			if err != nil {
				logs.Errorf("device config parse error: %v", err)
			}
		}
		if str, ok := data["createId"]; ok {
			device.CreateId, err = util.StringToInt64(str)
			if err != nil {
				logs.Errorf("device createId parse error: %v", err)
			}
		}
		if device.Id == "" {
			return nil
		}
		m.cache.Store(m.getDeviceKey(deviceId), device)
		return device
	}
	return nil
}

func (m *redisDeviceStore) PutDevice(device *core.Device) error {
	if device == nil {
		return errors.New("device not be nil")
	}
	if len(device.Id) == 0 {
		return errors.New("device id must be present")
	}
	rdb := redis.GetRedisClient()
	if rdb == nil {
		return errors.New("redis client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	data := map[string]string{
		"id":        device.Id,
		"name":      device.Name,
		"productId": device.ProductId,
		"devType":   device.DeviceType,
		"parentId":  device.ParentId,
		"createId":  fmt.Sprintf("%v", device.CreateId),
		"clusterId": device.ClusterId,
	}
	if device.Config != nil {
		b, err := json.Marshal(device.Config)
		if err != nil {
			return fmt.Errorf("device config marshal error: %w", err)
		}
		data["config"] = string(b)
	}
	err := rdb.HSet(ctx, m.getDeviceKey(device.Id), data).Err()
	if err != nil {
		return fmt.Errorf("hset device error: %w", err)
	}
	m.cache.Store(m.getDeviceKey(device.Id), device)
	return nil
}

func (m *redisDeviceStore) DelDevice(deviceId string) {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	deviceKey := m.getDeviceKey(deviceId)
	rdb.Del(ctx, deviceKey)
	m.cache.Delete(deviceKey)
}

func (m *redisDeviceStore) GetDeviceData(deviceId, key string) string {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	v, err := rdb.HGet(ctx, m.getDeviceKey(deviceId), "data:"+key).Result()
	if err != nil && err != redis.Nil {
		logs.Errorf("hget error: %v", err)
	}
	return v
}

func (m *redisDeviceStore) SetDeviceData(deviceId, key string, val any) {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := rdb.HSet(ctx, m.getDeviceKey(deviceId), "data:"+key, val).Err()
	if err != nil {
		logs.Errorf("hset error: %v", err)
	}
}

func (m *redisDeviceStore) updateClusterId(deviceId string) {
	localID := cluster.GetClusterId()
	rdb := redis.GetRedisClient()
	if rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := rdb.HSet(ctx, m.getDeviceKey(deviceId), "clusterId", localID).Err()
	if err != nil {
		logs.Errorf("updateClusterId error: %v", err)
	}
	device, ok := m.getDevice(deviceId)
	if ok && device != nil {
		device.ClusterId = localID
	}
}

// clearClusterIdIfOwner 设备离线时清空会话归属，便于 Gateway 走离线队列而非错误节点。
func (m *redisDeviceStore) clearClusterIdIfOwner(deviceId string) {
	localID := cluster.GetClusterId()
	// 先看内存缓存
	if device, ok := m.getDevice(deviceId); ok && device != nil {
		if len(device.ClusterId) > 0 && device.ClusterId != localID {
			return
		}
	} else {
		// 缓存未命中时读 Redis，避免误清
		dev := m.GetDevice(deviceId)
		if dev != nil && len(dev.ClusterId) > 0 && dev.ClusterId != localID {
			return
		}
	}

	rdb := redis.GetRedisClient()
	if rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	key := m.getDeviceKey(deviceId)
	// 仅当 redis 中仍是本节点时清空（防并发重连）
	cur, err := rdb.HGet(ctx, key, "clusterId").Result()
	if err != nil && err != redis.Nil {
		logs.Errorf("clearClusterId HGet error: %v", err)
		return
	}
	if len(cur) > 0 && cur != localID {
		return
	}
	if err := rdb.HSet(ctx, key, "clusterId", "").Err(); err != nil {
		logs.Errorf("clearClusterId HSet error: %v", err)
		return
	}
	if device, ok := m.getDevice(deviceId); ok && device != nil {
		device.ClusterId = ""
	}
	// 离线后去掉心跳超时探测成员，避免误踢
	_ = rdb.ZRem(ctx, m.getZKey(), deviceId).Err()
}
func (m *redisDeviceStore) getZKey() string {
	return "goiot:device_offline_check"
}

// RefreshOfflineTimeout 刷新设备过期时间
func (m *redisDeviceStore) RefreshOfflineTimeout(deviceId string) {
	device := m.GetDevice(deviceId)
	if device == nil {
		return
	}
	timeoutStr := device.GetConfig(core.DEVICE_TIMEOUT_KEY)
	if len(timeoutStr) > 0 {
		timeout, err := strconv.Atoi(timeoutStr)
		if err == nil && timeout > 0 {
			rdb := redis.GetRedisClient()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			// 使用 ZSet 存储过期时间，score 为过期时间戳
			expireTime := time.Now().Add(time.Duration(timeout) * time.Second).Unix()
			zKey := m.getZKey()
			err = rdb.ZAdd(ctx, zKey, &redis.Z{
				Score:  float64(expireTime),
				Member: deviceId,
			}).Err()
			if err != nil {
				logs.Errorf("refreshOfflineTimeout ZAdd error: %v", err)
			}
		}
	}
}

func (m *redisDeviceStore) startOfflineScanner() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		m.scanOfflineDevices()
	}
}

func (m *redisDeviceStore) scanOfflineDevices() {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	now := time.Now().Unix()
	// 获取所有过期设备
	zKey := m.getZKey()
	vals, err := rdb.ZRangeByScore(ctx, zKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", now),
	}).Result()

	if err != nil {
		logs.Errorf("scanOfflineDevices ZRangeByScore error: %v", err)
		return
	}

	if len(vals) > 0 {
		for _, deviceId := range vals {
			device := m.GetDevice(deviceId)
			if device != nil {
				session := core.GetSession(deviceId)
				if session != nil {
					session.Close()
				}
				core.DelSessionWithOfflineReason(deviceId, "heartbeat_timeout")
			}
			// 移除已处理的设备
			_, err := rdb.ZRem(ctx, zKey, deviceId).Result()
			if err != nil {
				logs.Errorf("%v", err)
			}
		}
	}
}

// product
func (p *redisDeviceStore) getProductKey(productId string) string {
	return "goiot:product:" + productId
}

func (m *redisDeviceStore) getProduct(productId string) (*core.Product, bool) {
	product, ok := m.cache.Load(m.getProductKey(productId))
	if ok {
		if product != nil {
			return product.(*core.Product), true
		}
		return nil, true
	}
	return nil, false
}

func (m *redisDeviceStore) GetProduct(productId string) *core.Product {
	product, ok := m.getProduct(productId)
	if ok {
		return product
	}
	if product == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		rdb := redis.GetRedisClient()
		productKey := m.getProductKey(productId)
		data, err := rdb.HGetAll(ctx, productKey).Result()
		if err != nil {
			if err != redis.Nil {
				logs.Errorf("hgetall proruct error: %v", err)
			}
			return nil
		}
		if len(data) == 0 {
			m.cache.Store(productKey, nil)
			return nil
		}
		product, err := core.NewProduct(data["id"], map[string]string{}, data["storePolicy"], data["tslData"])
		if err != nil {
			logs.Errorf("new product error: %v", err)
			return nil
		}
		product.NetworkType = data["networkType"]
		if str, ok := data["config"]; ok {
			err = json.Unmarshal([]byte(str), &product.Config)
			if err != nil {
				logs.Errorf("device config parse error: %v", err)
				return nil
			}
		}
		m.cache.Store(productKey, product)
		return product
	}
	return nil
}

// 保存产品
func (m *redisDeviceStore) PutProduct(product *core.Product) error {
	if product == nil {
		return errors.New("product not be nil")
	}
	if len(product.GetId()) == 0 {
		return errors.New("product id must be present")
	}
	data := map[string]string{
		"id":          product.Id,
		"storePolicy": product.StorePolicy,
		"tslData":     product.TslData.Text,
		"networkType": product.NetworkType,
	}
	if product.Config != nil {
		b, err := json.Marshal(product.Config)
		if err != nil {
			return fmt.Errorf("product config marshal error: %w", err)
		}
		data["config"] = string(b)
	}
	rdb := redis.GetRedisClient()
	if rdb == nil {
		return errors.New("redis client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	productKey := m.getProductKey(product.Id)
	err := rdb.HSet(ctx, productKey, data).Err()
	if err != nil {
		logs.Errorf("put product error: %v", err)
		return fmt.Errorf("hset product error: %w", err)
	}
	m.cache.Store(productKey, product)
	return nil
}

func (m *redisDeviceStore) DelProduct(productId string) {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	productId = m.getProductKey(productId)
	rdb.Del(ctx, productId)
	m.cache.Delete(productId)
}
