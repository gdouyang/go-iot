package mqttserver

import (
	"crypto/tls"
	"fmt"
	"go-iot/codec"
	"go-iot/network/servers"
	"net"
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"github.com/eclipse/paho.mqtt.golang/packets"
)

func init() {
	servers.RegServer(func() codec.NetServer {
		return NewServer()
	})
}

var m = map[string]*Broker{}

type (
	// Broker is MQTT server, will manage client, topic, session, etc.
	Broker struct {
		sync.RWMutex
		productId string
		name      string
		spec      *MQTTServerSpec

		listener net.Listener
		clients  map[string]*Client
		tlsCfg   *tls.Config

		// done is the channel for shutdowning this proxy.
		done chan struct{}
	}
)

func NewServer() *Broker {
	return &Broker{}
}

func (s *Broker) Type() codec.NetServerType {
	return codec.MQTT_BROKER
}

func (s *Broker) Start(network codec.NetworkConf) error {
	spec := &MQTTServerSpec{}
	spec.FromJson(network.Configuration)
	spec.Port = network.Port

	s.productId = network.ProductId
	s.name = spec.Name
	s.spec = spec
	s.clients = make(map[string]*Client)
	s.done = make(chan struct{})

	err := s.setListener()
	if err != nil {
		logs.Error("mqtt broker set listener failed: %v", err)
		return err
	}

	// create codec
	codec.NewCodec(network)

	go s.run()

	m[spec.Name] = s
	return nil
}

func (b *Broker) setListener() error {
	var l net.Listener
	var err error
	var cfg *tls.Config
	addr := fmt.Sprintf(":%d", b.spec.Port)
	if b.spec.UseTLS {
		cfg, err = b.spec.TlsConfig()
		if err != nil {
			return fmt.Errorf("invalid tls config for mqtt proxy: %v", err)
		}
		l, err = tls.Listen("tcp", addr, cfg)
		if err != nil {
			return fmt.Errorf("gen mqtt tls tcp listener with addr %v and cfg %v failed: %v", addr, cfg, err)
		}
	} else {
		l, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("gen mqtt tcp listener with addr %s failed: %v", addr, err)
		}
	}
	b.tlsCfg = cfg
	b.listener = l
	return err
}

func (b *Broker) Reload() error {
	return nil
}

func (b *Broker) Stop() error {
	close(b.done)
	b.listener.Close()

	b.Lock()
	defer b.Unlock()
	for _, v := range b.clients {
		go v.close()
	}
	b.clients = nil
	return nil
}

func (b *Broker) deleteSession(clientID string) {
	b.Lock()
	defer b.Unlock()
	if c, ok := b.clients[clientID]; ok {
		if !c.disconnected() {
			logs.Debug("broker watch and delete client %v", c.info.cid)
			c.close()
		}
	}
	delete(b.clients, clientID)
}

func (b *Broker) run() {
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			select {
			case <-b.done:
				return
			default:
			}
		} else {
			go b.handleConn(conn)
		}
	}
}

func (b *Broker) checkConnectPermission(connect *packets.ConnectPacket) bool {
	// check here to do early stop for connection. Later we will check it again to make sure
	// not exceed MaxAllowedConnection
	if b.spec.MaxAllowedConnection > 0 {
		b.Lock()
		connNum := len(b.clients)
		b.Unlock()
		if connNum >= b.spec.MaxAllowedConnection {
			return false
		}
	}
	return true
}

func (b *Broker) connectionValidation(connect *packets.ConnectPacket, conn net.Conn) (*Client, *packets.ConnackPacket, bool) {
	connack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	connack.SessionPresent = connect.CleanSession
	connack.ReturnCode = connect.Validate()
	if connack.ReturnCode != packets.Accepted {
		err := connack.Write(conn)
		logs.Error("invalid connection %v, write connack failed: %s", connack.ReturnCode, err)
		return nil, nil, false
	}
	// check rate limiter and max allowed connection
	if !b.checkConnectPermission(connect) {
		logs.Debug("client %v not get connect permission from rate limiter", connect.ClientIdentifier)
		connack.ReturnCode = packets.ErrRefusedServerUnavailable
		err := connack.Write(conn)
		if err != nil {
			logs.Error("connack back to client %s failed: %s", connect.ClientIdentifier, err)
		}
		return nil, nil, false
	}

	client := newClient(connect, b, conn)

	// check auth
	ctx := &authContext{
		BaseContext: codec.BaseContext{
			ProductId: b.productId,
			Session:   nil,
		},
		client:  client,
		connack: connack,
		conn:    conn,
	}
	err := codec.GetCodec(b.productId).OnConnect(ctx)

	if ctx.authFail {
		return nil, nil, false
	}
	if err != nil {
		if err.Error() == "notimpl" && !ctx.checkAuth() {
			return nil, nil, false
		}
		logs.Error(err)
		return nil, nil, false
	}

	return client, connack, true
}

func (b *Broker) handleConn(conn net.Conn) {
	defer conn.Close()
	packet, err := packets.ReadPacket(conn)
	if err != nil {
		logs.Error("read connect packet failed: %s", err)
		return
	}
	connect, ok := packet.(*packets.ConnectPacket)
	if !ok {
		logs.Error("first packet received %s that was not Connect", packet.String())
		return
	}
	logs.Debug("connection from client %s", connect.ClientIdentifier)

	client, connack, valid := b.connectionValidation(connect, conn)
	if !valid {
		return
	}

	cid := client.info.cid

	b.Lock()
	if oldClient, ok := b.clients[cid]; ok {
		logs.Debug("client %v take over by new client with same name", oldClient.info.cid)
		go oldClient.close()
	}
	b.clients[client.info.cid] = client
	b.Unlock()

	b.setSession(client, connect)

	err = connack.Write(conn)
	if err != nil {
		logs.Error("send connack to client %s failed: %s", connect.ClientIdentifier, err)
		client.close()
		return
	}

	go client.writeLoop()
	client.readLoop()
}

func (b *Broker) setSession(client *Client, connect *packets.ConnectPacket) {
	// when clean session is false, previous session exist and previous session not clean session,
	// then we use previous session, otherwise use new session
	prevS := codec.GetSessionManager().Get(client.info.deviceId)
	var prevSess *Session = nil
	if prevS != nil {
		prevSess = prevS.(*Session)
	}
	if !connect.CleanSession && (prevSess != nil) && !prevSess.cleanSession() {
		client.session = prevSess
	} else {
		if prevSess != nil {
			prevSess.Disconnect()
		}
		sess := &Session{}
		sess.init(b, connect)
		// here connect is valid, make device online
		baseContext := &codec.BaseContext{
			ProductId: b.productId,
			Session:   sess,
		}
		baseContext.DeviceOnline(client.info.deviceId)
		client.session = sess
	}
}

func (b *Broker) getClient(clientID string) *Client {
	b.RLock()
	defer b.RUnlock()
	if val, ok := b.clients[clientID]; ok {
		return val
	}
	return nil
}

func (b *Broker) removeClient(clientID string) {
	b.Lock()
	if val, ok := b.clients[clientID]; ok {
		if val.disconnected() {
			delete(b.clients, clientID)
		}
	}
	b.Unlock()
}

func (b *Broker) TotalConnection() int32 {
	l := len(b.clients)
	return int32(l)
}
