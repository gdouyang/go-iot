package mqtt5

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"go-iot/pkg/option"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	logs "go-iot/pkg/logger"

	mqtt "github.com/mochi-mqtt/server/v2"
	listeners "github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

var broker = Broker{}

type (
	// Broker is MQTT5 server, manages clients, topics, sessions, etc.
	Broker struct {
		sync.RWMutex
		name    string
		spec    *MQTTServerSpec
		server  *mqtt.Server
		tlsCfg  *tls.Config
		clients map[string]*ClientAndSession
	}
	// 事件钩子
	BrokerHook struct {
		mqtt.HookBase
		broker *Broker
	}
	// 事件钩子选项
	BrokerHookOptions struct {
		broker *Broker
	}
)

func (s *Broker) Type() network.NetType {
	return network.GOIOT_MQTT_BROKER
}

func Start() error {
	var certs []network.Certificate
	for _, v := range option.Global.Mqtt.Certificate {
		certs = append(certs, network.Certificate{
			Name: v.Name,
			Cert: v.Cert,
			Key:  v.Key,
		})
	}
	var spec = &MQTTServerSpec{
		Host:        option.Global.Mqtt.Host,
		Name:        option.Global.Mqtt.Name,
		Port:        int32(option.Global.Mqtt.Port),
		UseTLS:      option.Global.Mqtt.UseTLS,
		Certificate: certs,
	}
	s := &broker
	s.name = spec.Name
	s.spec = spec
	s.clients = make(map[string]*ClientAndSession)

	// Configure TLS if enabled
	return s.init(spec)
}

// Stop shuts down the built-in MQTT5 broker (clients + server).
func Stop() error {
	return broker.Stop()
}

func (b *Broker) Stop() error {
	var clients []*ClientAndSession
	func() {
		b.Lock()
		defer b.Unlock()
		clients = make([]*ClientAndSession, 0, len(b.clients))
		for _, v := range b.clients {
			clients = append(clients, v)
		}
		b.clients = make(map[string]*ClientAndSession)
	}()

	// 限流并发关闭：全量 go 会瞬间产生 10 万 goroutine 并在 mochi clients map 上激烈竞争；
	// 串行关闭又太慢（每个 Close 含 DisconnectClient）。用信号量限制同时关闭数。
	const closeConcurrency = 2000
	sem := make(chan struct{}, closeConcurrency)
	var wg sync.WaitGroup
	for _, v := range clients {
		wg.Add(1)
		go func(c *ClientAndSession) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c.Close()
			// Stop 已清空 clients map，后续 OnDisconnect 拿不到 client 对象，
			// 在此补 session 下线（幂等，OnDisconnect 已处理时无副作用）
			core.DelSessionWithTimeoutCheck(c.info.deviceId)
		}(v)
	}
	wg.Wait()

	// server.Close 必须在锁外：mochi CloseAll 会等待 Serve goroutine 退出，
	// 而它可能正阻塞在 OnDisconnect 的 b.Lock 上（同 clientID 重复连接路径），持锁等待会死锁
	if b.server != nil {
		err := b.server.Close()
		b.server = nil
		return err
	}
	return nil
}

func (s *Broker) init(spec *MQTTServerSpec) error {
	// Create the new MQTT Server.
	var capabilities = mqtt.NewDefaultServerCapabilities()
	// 大连接量场景下调小每连接缓冲，降低内存占用（10万连接约省700MB）
	// 压测场景以 QoS0 上报为主，128 深度写队列足够；QoS1/2 大批量下发时再调大
	capabilities.MaximumClientWritesPending = 128
	capabilities.MaximumInflight = 128
	server := mqtt.New(&mqtt.Options{
		Logger:       slog.New(logs.NewSugaredHandler()),
		Capabilities: capabilities,
		// ClientNetWriteBufferSize: 4096,
		// ClientNetReadBufferSize:  4096,
		// SysTopicResendInterval: 10,
	})

	if s.spec.UseTLS {
		cfg, err := s.spec.TlsConfig()
		if err != nil {
			return fmt.Errorf("invalid tls config for mqtt proxy: %v", err)
		}
		s.tlsCfg = cfg
	}

	// Create a TCP listener on a standard port.
	config := listeners.Config{ID: "goiot-mqtt-broker", Address: fmt.Sprintf(":%d", spec.Port)}
	if s.spec.UseTLS {
		config.TLSConfig = s.tlsCfg
	}
	tcp := listeners.NewTCP(config)
	err := server.AddListener(tcp)
	if err != nil {
		return err
	}
	// 给broker增加Hook
	err = server.AddHook(&BrokerHook{}, &BrokerHookOptions{broker: s})
	if err != nil {
		return err
	}

	// Start the server
	go func() {
		if rec := recover(); rec != nil {
			l := fmt.Sprintf("mqtt broker panic: %v \n %s", rec, string(debug.Stack()))
			logs.Errorf(l)
		}
		err := server.Serve()
		if err != nil {
			logs.Errorf("MQTT server error: %v", err)
		}
	}()

	s.server = server
	return nil
}

func (b *Broker) TotalConnection() int32 {
	b.RLock()
	defer b.RUnlock()
	return int32(len(b.clients))
}

func (h *BrokerHook) ID() string {
	return "goiot-mqtt-broker-hook"
}

func (h *BrokerHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnectAuthenticate,
		mqtt.OnACLCheck,
		mqtt.OnDisconnect,
		mqtt.OnSubscribed,
		mqtt.OnUnsubscribed,
		mqtt.OnPublished,
		// mqtt.OnPublish,
	}, []byte{b})
}

func (h *BrokerHook) Init(config any) error {
	logs.Debugf("initialised")
	opt := config.(*BrokerHookOptions)
	h.broker = opt.broker
	return nil
}

// 当用户尝试与服务器进行身份验证时调用。
// 必须实现此方法来允许或拒绝对服务器的访问（请参阅 hooks/auth/allow_all 或 basic）。
// 它可以在自定义Hook钩子中使用，以检查连接的用户是否与现有用户数据库中的用户匹配。
// 如果允许访问，则返回 true。
func (h *BrokerHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	logs.Debugf("client connected %s", cl.ID)
	h.broker.Lock()
	defer h.broker.Unlock()
	client := NewClient(cl, h.broker)
	client.info.password = string(pk.Connect.Password)

	// find product by clientId (assuming clientId is deviceId)
	deviceId := cl.ID
	dev := core.GetDevice(deviceId)
	if dev == nil {
		// try to find in db
		logs.Errorf("device not found: %s", deviceId)
		return false
	}

	productId := dev.ProductId
	client.info.productId = productId
	client.info.deviceId = deviceId

	// check auth
	ctx := &authContext{
		BaseContext: core.BaseContext{
			ProductId: productId,
			Session:   nil,
			DeviceId:  deviceId,
		},
		client: client,
	}
	codec := core.GetCodec(productId)
	if codec == nil {
		logs.Errorf("codec not found for product: %s", productId)
		return false
	}
	err := codec.OnConnect(ctx)
	if ctx.authFailCode != 0 {
		return false
	}
	if err != nil {
		if err != core.ErrFunctionNotImpl {
			logs.Errorf(err.Error())
			return false
		}
		if !ctx.checkAuth() {
			return false
		}
		if err := ctx.DeviceOnline(ctx.DeviceId); err != nil {
			logs.Errorf("mqtt5 DeviceOnline error: %v", err)
			return false
		}
	}
	h.broker.clients[cl.ID] = client
	return true
}

// 当用户尝试发布或订阅主题时调用，用来检测ACL规则。
func (h *BrokerHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	return true
}

// 当客户端因任何原因断开连接时调用。
func (h *BrokerHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	if err != nil {
		logs.Debugf("client disconnected %s expire: %v error: %v", cl.ID, expire, err)
	} else {
		logs.Debugf("client disconnected %s expire: %v", cl.ID, expire)
	}
	// 有限重试获取锁：锁内只有 map 操作（微秒级），正常竞争窗口极短；
	// 持锁方异常卡死（如认证脚本死循环）时有限重试后放弃，保证断开处理不阻塞系统
	// （残留条目/session 由同 ID 重连覆盖或 Stop 兜底）
	for i := 0; i < 100; i++ {
		if h.broker.TryLock() {
			// 锁内只做条目删除（微秒级）；closeByBroker/DelSession 等慢操作全部移到锁外
			var client *ClientAndSession
			logs.Debugf("delete client: %s", cl.ID)
			if c := h.broker.clients[cl.ID]; c != nil && c.client == cl {
				// 只清理属于本次断开的连接：同 clientID 抢占（session takeover）时，新连接
				// 的 OnConnectAuthenticate 已把 clients[id] 替换为新条目，若按 ID 清理会
				// 误删新连接条目并下线其 session（新连接"失明"、上报静默丢弃）
				client = c
				delete(h.broker.clients, cl.ID)
			}
			h.broker.Unlock()
			if client != nil {
				// closeByBroker 幂等（TryLock + isClose 判断）；DelSession 不能放在 !isClose 内：
				// Stop 场景 v.Close() 先置 isClose，后触发的 OnDisconnect 会跳过导致 session 残留
				client.closeByBroker()
				core.DelSessionWithTimeoutCheck(client.info.deviceId)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	// 放弃：持锁方异常卡死（如认证脚本死循环），系统保活优先
	logs.Warnf("mqtt5 OnDisconnect give up lock for client %s after 100ms", cl.ID)
}

// 当客户端成功订阅一个或多个主题时调用。
func (h *BrokerHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte) {
	logs.Debugf("subscribed qos=%v client=%s filters=%v", reasonCodes, cl.ID, pk.Filters)
}

// 当客户端成功取消订阅一个或多个主题时调用。
func (h *BrokerHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet) {
	logs.Debugf("unsubscribed client: %s filters: %v", cl.ID, pk.Filters)
}

// 当客户端向订阅者发布消息后调用
func (h *BrokerHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	logs.Debugf("published to client: %s payload: %s", cl.ID, string(pk.Payload))

	// RLock 保护 clients map 读取（与 OnConnectAuthenticate/OnDisconnect 的写入并发时
	// 无锁读会触发 fatal: concurrent map read and map write）
	h.broker.RLock()
	c := h.broker.clients[cl.ID]
	h.broker.RUnlock()
	if c == nil {
		// client might be disconnected or not yet in map
		return
	}
	// 调用编解码脚本处理
	sc := core.GetCodec(c.info.productId)
	if sc == nil {
		logs.Warnf("codec not found for product: %s", c.info.productId)
		return
	}
	sc.OnMessage(&mqttContext{
		BaseContext: core.BaseContext{
			DeviceId:  c.GetDeviceId(),
			ProductId: c.info.productId,
			Session:   c,
		},
		Data:      pk.Payload,
		topic:     pk.TopicName,
		messageID: pk.PacketID,
		client:    c,
	})
}
