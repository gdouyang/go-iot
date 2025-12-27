package store

import (
	"context"
	"encoding/json"
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

func (m *redisDeviceStore) PutDevice(device *core.Device) {
	rdb := redis.GetRedisClient()
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
			panic(err)
		}
		data["config"] = string(b)
	}
	err := rdb.HSet(ctx, m.getDeviceKey(device.Id), data).Err()
	if err != nil {
		panic(fmt.Errorf("hset error: %v", err))
	}
	m.cache.Store(m.getDeviceKey(device.Id), device)
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
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := rdb.HSet(ctx, m.getDeviceKey(deviceId), "clusterId", cluster.GetClusterId()).Err()
	if err != nil {
		logs.Errorf("updateClusterId error: %v", err)
	}
	device, ok := m.getDevice(deviceId)
	if ok {
		device.ClusterId = cluster.GetClusterId()
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
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
				core.DelSessionWithOfflineReason(deviceId, "heartbeat_timeout")
			}
			// 移除已处理的设备
			rdb.ZRem(ctx, zKey, deviceId)
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
func (m *redisDeviceStore) PutProduct(product *core.Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id must be present")
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
			panic(err)
		}
		data["config"] = string(b)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rdb := redis.GetRedisClient()
	productKey := m.getProductKey(product.Id)
	err := rdb.HSet(ctx, productKey, data).Err()
	if err != nil {
		logs.Errorf("put product error: %v", err)
		panic(err)
	}
	m.cache.Store(productKey, product)
}

func (m *redisDeviceStore) DelProduct(productId string) {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	productId = m.getProductKey(productId)
	rdb.Del(ctx, productId)
	m.cache.Delete(productId)
}
