/*
 * Copyright (c) 2017, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mqttproxy

import (
	"errors"
	"go-iot/provider/codec"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/beego/beego/v2/core/logs"
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
	"*packets.PublishPacket": func(c *Client, packet packets.ControlPacket) error {
		publish := packet.(*packets.PublishPacket)
		logs.Debug("client %s process publish %v", c.info.cid, publish.TopicName)
		return pipelineWrapper(processPublish, Publish)(c, packet)
	},
}

type (
	// ClientInfo is basic information for client
	ClientInfo struct {
		cid       string
		username  string
		password  string
		keepalive uint16
		will      *packets.PublishPacket
	}

	// Client represents a MQTT client connection in Broker
	Client struct {
		sync.Mutex

		broker  *Broker
		session *Session
		conn    net.Conn

		info       ClientInfo
		statusFlag int32
		writeCh    chan packets.ControlPacket
		done       chan struct{}

		// kv map is used for pipeline to share messages among filters during whole connection
		kvMap sync.Map
	}
)

// var _ context.MQTTClient = (*Client)(nil)

// ClientID return client id of Client
func (c *Client) ClientID() string {
	return c.info.cid
}

// Load load value keep in Client kv map
func (c *Client) Load(key interface{}) (value interface{}, ok bool) {
	return c.kvMap.Load(key)
}

// Store store key-value pair in Client kv map
func (c *Client) Store(key interface{}, value interface{}) {
	c.kvMap.Store(key, value)
}

// Delete delete key-value pair in Client kv map
func (c *Client) Delete(key interface{}) {
	c.kvMap.Delete(key)
}

// UserName return username of Client
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
		c.closeAndDelSession()
		c.broker.removeClient(c.info.cid)
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
				logs.Error("set read timeout failed: %s", c.info.cid)
			}
		}

		logs.Debug("client %s readLoop read packet", c.info.cid)
		packet, err := packets.ReadPacket(c.conn)
		if err != nil {
			logs.Error("client %s read packet failed: %v", c.info.cid, err)
			return
		}
		if _, ok := packet.(*packets.DisconnectPacket); ok {
			c.info.will = nil
			return
		}
		err = c.processPacket(packet)
		if err != nil {
			logs.Error("client %s process packet failed: %v", c.info.cid, err)
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
				logs.Error("write packet %v to client %s failed: %s", p.String(), c.info.cid, err)
				c.closeAndDelSession()
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
	logs.Debug("client %v connection close", c.info.cid)
	atomic.StoreInt32(&c.statusFlag, Disconnected)
	close(c.done) // 删除
	// c.session.close()
	c.broker.deleteSession(c.info.cid)
	c.Unlock()

}

func (c *Client) disconnected() bool {
	return atomic.LoadInt32(&c.statusFlag) == Disconnected
}

func (c *Client) closeAndDelSession() {
	c.broker.sessMgr.delLocal(c.info.cid)

	c.close()
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
	switch publish.Qos {
	case QoS0:
		// do nothing
	case QoS1:
		puback := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
		puback.MessageID = publish.MessageID
		c.writePacket(puback)
	case QoS2:
		// not support yet
	}
	// 调用wasm host处理
	sc := codec.GetCodec(c.broker.productId)
	sc.Decode(&mqttContext{productId: c.broker.productId, Data: publish.Payload})
}

func processPuback(c *Client, packet packets.ControlPacket) {
	puback := packet.(*packets.PubackPacket)
	c.session.puback(puback)
}

func processSubscribe(c *Client, p packets.ControlPacket) {
	packet := p.(*packets.SubscribePacket)
	logs.Debug("client %s subscribe %v with qos %v", c.info.cid, packet.Topics, packet.Qoss)

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

	logs.Debug("client %s processUnsubscribe %v", c.info.cid, packet.Topics)

	c.session.unsubscribe(packet.Topics)

	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID
	c.writePacket(unsuback)
}

func processPingreq(c *Client, packet packets.ControlPacket) {
	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	c.writePacket(resp)
}
