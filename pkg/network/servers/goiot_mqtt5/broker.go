package goiot_mqtt5

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
	b.Lock()
	defer b.Unlock()
	for _, v := range b.clients {
		go v.Close()
	}
	b.clients = make(map[string]*ClientAndSession)
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
	capabilities.MaximumClientWritesPending = 1024
	capabilities.MaximumInflight = 1024
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
			logs.Errorf("goiot_mqtt5 DeviceOnline error: %v", err)
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
	var getLock = h.broker.TryLock()
	var unlock = func() {
		if getLock {
			h.broker.Unlock()
		}
	}
	defer unlock()
	logs.Debugf("delete client: %s", cl.ID)
	var client = h.broker.clients[cl.ID]
	if client != nil && !client.isClose {
		client.Close()
		core.DelSessionWithTimeoutCheck(client.info.deviceId)
	}
	delete(h.broker.clients, cl.ID)
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

	c := h.broker.clients[cl.ID]
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
