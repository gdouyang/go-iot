package mqtt5

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"go-iot/pkg/network/servers"
	"sync"

	logs "go-iot/pkg/logger"

	mqtt "github.com/mochi-mqtt/server/v2"
	listeners "github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

func init() {
	servers.RegServer(func() network.NetServer {
		return NewServer()
	})
}

var m = map[string]*Broker{}

type (
	// Broker is MQTT5 server, manages clients, topics, sessions, etc.
	Broker struct {
		sync.RWMutex
		productId string
		name      string
		spec      *MQTTServerSpec
		server    *mqtt.Server
		tlsCfg    *tls.Config
		clients   map[string]*ClientAndSession

		done chan struct{}
	}
	// 事件钩子
	BrokerHook struct {
		mqtt.HookBase
		productId string
		broker    *Broker
	}
	// 事件钩子选项
	BrokerHookOptions struct {
		broker *Broker
	}
)

func NewServer() *Broker {
	return &Broker{}
}

func (s *Broker) Type() network.NetType {
	return network.MQTT_BROKER
}

func (s *Broker) Start(network network.NetworkConf) error {
	spec := &MQTTServerSpec{}
	err := spec.FromNetwork(network)
	if err != nil {
		return err
	}
	spec.Port = network.Port

	s.productId = network.ProductId
	s.name = spec.Name
	s.spec = spec
	s.clients = make(map[string]*ClientAndSession)
	s.done = make(chan struct{})

	// Create the new MQTT Server.
	server := mqtt.New(nil)

	// Configure TLS if enabled
	if s.spec.UseTLS {
		cfg, err := s.spec.TlsConfig()
		if err != nil {
			return fmt.Errorf("invalid tls config for mqtt proxy: %v", err)
		}
		s.tlsCfg = cfg
	}

	// Create a TCP listener on a standard port.
	config := listeners.Config{ID: "mqtt-" + s.productId, Address: fmt.Sprintf(":%d", spec.Port)}
	if s.spec.UseTLS {
		config.TLSConfig = s.tlsCfg
	}
	tcp := listeners.NewTCP(config)
	err = server.AddListener(tcp)
	if err != nil {
		return err
	}

	// 给broker增加Hook
	err = server.AddHook(new(BrokerHook), &BrokerHookOptions{broker: s})
	if err != nil {
		return err
	}

	// Start the server
	go func() {
		err := server.Serve()
		if err != nil {
			logs.Errorf("MQTT server error: %v", err)
		}
	}()

	s.server = server
	m[spec.Name] = s
	return nil
}

func (b *Broker) Stop() error {

	close(b.done)
	b.Lock()
	defer b.Unlock()
	if b.server != nil {
		b.server.Close()
	}
	for _, v := range b.clients {
		go v.Close()
	}
	b.clients = nil
	return nil
}

func (b *Broker) Reload() error {
	return nil
}

func (b *Broker) TotalConnection() int32 {
	b.RLock()
	defer b.RUnlock()
	return int32(len(b.clients))
}

func (h *BrokerHook) ID() string {
	return "mqtt-" + h.productId
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
	h.productId = opt.broker.productId
	return nil
}

// 当用户尝试与服务器进行身份验证时调用。
// 必须实现此方法来允许或拒绝对服务器的访问（请参阅 hooks/auth/allow_all 或 basic）。
// 它可以在自定义Hook钩子中使用，以检查连接的用户是否与现有用户数据库中的用户匹配。
// 如果允许访问，则返回 true。
func (h *BrokerHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	logs.Debugf("client connected %s", cl.ID)
	client := NewClient(cl, h.broker)
	// check auth
	ctx := &authContext{
		BaseContext: core.BaseContext{
			ProductId: h.productId,
			Session:   nil,
			DeviceId:  client.ClientID(),
		},
		client: client,
	}
	err := core.GetCodec(h.productId).OnConnect(ctx)
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
		ctx.DeviceOnline(ctx.DeviceId)
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
	var client = h.broker.clients[cl.ID]
	if client != nil {
		client.Close()
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
	// 调用编解码脚本处理
	sc := core.GetCodec(h.productId)
	sc.OnMessage(&mqttContext{
		BaseContext: core.BaseContext{
			DeviceId:  c.GetDeviceId(),
			ProductId: c.broker.productId,
			Session:   c,
		},
		Data:      pk.Payload,
		topic:     pk.TopicName,
		messageID: pk.PacketID,
	})
}
