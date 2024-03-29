package mqttserver

import (
	"errors"
	"go-iot/pkg/core"
	"io"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	logs "go-iot/pkg/logger"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

const (
	// Connected is MQTT client status of Connected
	Connected = 1
	// Disconnected is MQTT client status of Disconnected
	Disconnected = 2

	// QoS0 for "At most once"
	QoS0 byte = 0
	// QoS1 for "At least once
	QoS1 byte = 1
	// QoS2 for "Exactly once"
	QoS2 byte = 2
)

type processFn func(*Client, packets.ControlPacket)
type processFnWithErr func(*Client, packets.ControlPacket) error

var processPacketMap = map[string]processFnWithErr{
	"*packets.ConnectPacket":     errorWrapper("double connect"),
	"*packets.ConnackPacket":     errorWrapper("client should not send connack"),
	"*packets.PubrecPacket":      errorWrapper("qos2 not support now"),
	"*packets.PubrelPacket":      errorWrapper("qos2 not support now"),
	"*packets.PubcompPacket":     errorWrapper("qos2 not support now"),
	"*packets.SubackPacket":      errorWrapper("broker not subscribe"),
	"*packets.UnsubackPacket":    errorWrapper("broker not unsubscribe"),
	"*packets.PingrespPacket":    errorWrapper("broker not ping"),
	"*packets.SubscribePacket":   pipelineWrapper(processSubscribe, Subscribe),
	"*packets.UnsubscribePacket": pipelineWrapper(processUnsubscribe, Unsubscribe),
	"*packets.PingreqPacket":     nilErrWrapper(processPingreq),
	"*packets.PubackPacket":      nilErrWrapper(processPuback),
	"*packets.PublishPacket":     pipelineWrapper(processPublish, Publish),
}

type (
	// ClientInfo is basic information for client
	ClientInfo struct {
		cid       string
		username  string
		password  string
		deviceId  string
		keepalive uint16
		will      *packets.PublishPacket
	}

	// Client represents a MQTT client connection in Broker
	Client struct {
		sync.Mutex

		broker  *Broker
		session *MqttSession
		conn    net.Conn

		info       ClientInfo
		statusFlag int32
		writeCh    chan packets.ControlPacket
		done       chan struct{}
	}
)

// client id of Client
func (c *Client) ClientID() string {
	return c.info.cid
}

// username of Client
func (c *Client) UserName() string {
	return c.info.username
}

func newClient(connect *packets.ConnectPacket, broker *Broker, conn net.Conn) *Client {
	var will *packets.PublishPacket
	if connect.WillFlag {
		will = packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
		will.Qos = connect.WillQos
		will.TopicName = connect.WillTopic
		will.Retain = connect.WillRetain
		will.Payload = connect.WillMessage
		will.Dup = connect.Dup
	}

	info := ClientInfo{
		cid:       connect.ClientIdentifier,
		username:  connect.Username,
		password:  string(connect.Password),
		keepalive: connect.Keepalive,
		will:      will,
	}
	client := &Client{
		broker:     broker,
		conn:       conn,
		info:       info,
		statusFlag: Connected,
		writeCh:    make(chan packets.ControlPacket, 50),
		done:       make(chan struct{}),
	}
	return client
}

func (c *Client) readLoop() {
	defer func() {
		c.close()
	}()
	keepAlive := time.Duration(c.info.keepalive) * time.Second
	timeOut := keepAlive + keepAlive/2
	for {
		select {
		case <-c.done:
			return
		default:
		}

		if keepAlive > 0 {
			if err := c.conn.SetDeadline(time.Now().Add(timeOut)); err != nil {
				logs.Errorf("set read timeout failed: %s", c.info.cid)
			}
		}

		logs.Debugf("client %s readLoop read packet", c.info.cid)
		packet, err := packets.ReadPacket(c.conn)
		if err != nil {
			if err != io.EOF {
				logs.Errorf("client %s read packet failed: %v", c.info.cid, err)
			}
			return
		}
		if _, ok := packet.(*packets.DisconnectPacket); ok {
			c.info.will = nil
			return
		}
		// 根据不同类型的包来选择不同的处理方式
		err = c.processPacket(packet)
		if err != nil {
			logs.Errorf("client %s process packet failed: %v", c.info.cid, err)
			return
		}
	}
}

func (c *Client) processPacket(packet packets.ControlPacket) error {
	packetType := reflect.TypeOf(packet).String()
	fn, ok := processPacketMap[packetType]
	if !ok {
		return errors.New("unknown packet")
	}
	return fn(c, packet)
}

func (c *Client) writePacket(packet packets.ControlPacket) {
	c.writeCh <- packet
}

func (c *Client) writeLoop() {
	for {
		select {
		case p := <-c.writeCh:
			err := p.Write(c.conn)
			if err != nil {
				logs.Errorf("write packet %v to client %s failed: %s", p.String(), c.info.cid, err)
				c.close()
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) close() {
	c.Lock()
	if c.disconnected() {
		c.Unlock()
		return
	}
	logs.Debugf("client %v connection close", c.info.cid)
	atomic.StoreInt32(&c.statusFlag, Disconnected)
	close(c.done) // 删除
	c.broker.deleteSession(c.info.cid)
	c.broker.removeClient(c.info.cid)
	c.conn.Close()
	c.Unlock()
	if c.session != nil {
		c.session.Disconnect()
	}
}

func (c *Client) disconnected() bool {
	return atomic.LoadInt32(&c.statusFlag) == Disconnected
}

func (c *Client) Done() <-chan struct{} {
	return c.done
}

func errorWrapper(errMsg string) processFnWithErr {
	return func(c *Client, p packets.ControlPacket) error {
		return errors.New(errMsg)
	}
}

func nilErrWrapper(fn processFn) processFnWithErr {
	return func(c *Client, p packets.ControlPacket) error {
		fn(c, p)
		return nil
	}
}

func pipelineWrapper(fn processFn, packetType PacketType) processFnWithErr {
	return func(c *Client, p packets.ControlPacket) error {
		fn(c, p)
		return nil
	}
}

func processPublish(c *Client, packet packets.ControlPacket) {
	publish := packet.(*packets.PublishPacket)
	logs.Debugf("client %s process publish %v", c.info.cid, publish.TopicName)
	switch publish.Qos {
	case QoS0:
		// do nothing
	case QoS1:
		puback := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
		puback.MessageID = publish.MessageID
		c.writePacket(puback) // 返回客户端ack
	case QoS2:
		// not support yet
	}
	// 调用wasm host处理
	sc := core.GetCodec(c.broker.productId)
	sc.OnMessage(&mqttContext{
		BaseContext: core.BaseContext{
			DeviceId:  c.session.GetDeviceId(),
			ProductId: c.broker.productId,
			Session:   c.session,
		},
		Data:      publish.Payload,
		topic:     publish.TopicName,
		messageID: publish.MessageID,
	})
}

func processPuback(c *Client, packet packets.ControlPacket) {
	puback := packet.(*packets.PubackPacket)
	c.session.puback(puback)
}

func processSubscribe(c *Client, p packets.ControlPacket) {
	packet := p.(*packets.SubscribePacket)
	logs.Debugf("client %s subscribe %v with qos %v", c.info.cid, packet.Topics, packet.Qoss)

	c.session.subscribe(packet.Topics, packet.Qoss)

	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	suback.ReturnCodes = make([]byte, len(packet.Topics))
	for i := range packet.Topics {
		suback.ReturnCodes[i] = packet.Qos
	}
	c.writePacket(suback)
}

func processUnsubscribe(c *Client, p packets.ControlPacket) {
	packet := p.(*packets.UnsubscribePacket)

	logs.Debugf("client %s processUnsubscribe %v", c.info.cid, packet.Topics)

	c.session.unsubscribe(packet.Topics)

	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID
	c.writePacket(unsuback)
}

func processPingreq(c *Client, packet packets.ControlPacket) {
	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	c.writePacket(resp)
}
