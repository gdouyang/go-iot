package store

import (
	"context"
	"encoding/json"
	"fmt"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/core/util"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/redis"
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
	eventbus.Subscribe(eventbus.GetOnlineTopic("*", "*"), func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OnlineMessage); ok {
			store.updateClusterId(m.DeviceId)
		}
	})
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
		data, err := rdb.HGetAll(ctx, m.getDeviceKey(deviceId)).Result()
		if err != nil {
			logs.Errorf("hgetall error: %v", err)
		}
		if len(data) == 0 {
			m.cache.Store(m.getDeviceKey(deviceId), nil)
			return nil
		}
		config := map[string]string{}
		if str, ok := data["config"]; ok {
			err = json.Unmarshal([]byte(str), &config)
			if err != nil {
				logs.Errorf("device config parse error: %v", err)
			}
		}
		dat := map[string]string{}
		if str, ok := data["data"]; ok {
			err = json.Unmarshal([]byte(str), &dat)
			if err != nil {
				logs.Errorf("device data parse error: %v", err)
			}
		}
		var createId int64
		if str, ok := data["createId"]; ok {
			createId, err = util.StringToInt64(str)
			if err != nil {
				logs.Errorf("device createId parse error: %v", err)
			}
		}
		dev := &core.Device{
			Id:         data["id"],
			ProductId:  data["productId"],
			CreateId:   createId,
			Config:     config,
			Data:       dat,
			ClusterId:  data["clusterId"],
			DeviceType: data["devType"],
			ParentId:   data["parentId"],
		}
		m.cache.Store(m.getDeviceKey(deviceId), dev)
		return dev
	}
	return nil
}

func (m *redisDeviceStore) PutDevice(device *core.Device) {
	p := device
	byt, _ := json.Marshal(p.Config)
	dat, _ := json.Marshal(p.Data)
	data := map[string]string{
		"id":        p.Id,
		"productId": p.ProductId,
		"devType":   p.DeviceType,
		"parentId":  p.ParentId,
		"createId":  fmt.Sprintf("%v", p.CreateId),
		"clusterId": p.ClusterId,
		"config":    string(byt),
		"data":      string(dat),
	}
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err := rdb.HSet(ctx, m.getDeviceKey(p.Id), data).Err()
	if err != nil {
		logs.Errorf("hset error: %v", err)
	}
	m.cache.Store(m.getDeviceKey(p.Id), device)
}

func (m *redisDeviceStore) DelDevice(deviceId string) {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	deviceId = m.getDeviceKey(deviceId)
	rdb.Del(ctx, deviceId)
	m.cache.Delete(deviceId)
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
		data, err := rdb.HGetAll(ctx, m.getProductKey(productId)).Result()
		if err != nil {
			logs.Errorf("get proruct error: %v", err)
		}
		if len(data) == 0 {
			m.cache.Store(m.getProductKey(productId), nil)
			return nil
		}
		config := map[string]string{}
		if str, ok := data["config"]; ok {
			err = json.Unmarshal([]byte(str), &config)
			if err != nil {
				logs.Errorf("device config parse error: %v", err)
			}
		}
		produ, err := core.NewProduct(data["id"], config, data["storePolicy"], data["tslData"])
		if err != nil {
			logs.Errorf("new product error: %v", err)
		} else {
			m.cache.Store(m.getProductKey(productId), produ)
			return produ
		}
	}
	return nil
}

func (m *redisDeviceStore) PutProduct(product *core.Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id must be present")
	}
	p := product
	byt, _ := json.Marshal(p.Config)
	data := map[string]string{
		"id":          p.Id,
		"storePolicy": p.StorePolicy,
		"config":      string(byt),
		"tslData":     p.TslData.Text,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rdb := redis.GetRedisClient()
	err := rdb.HSet(ctx, m.getProductKey(p.Id), data).Err()
	if err != nil {
		logs.Errorf("put product error: %v", err)
	}
	m.cache.Store(m.getProductKey(p.Id), product)
}

func (m *redisDeviceStore) DelProduct(productId string) {
	rdb := redis.GetRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	productId = m.getProductKey(productId)
	rdb.Del(ctx, productId)
	m.cache.Delete(productId)
}
