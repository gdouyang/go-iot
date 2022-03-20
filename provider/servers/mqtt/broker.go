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
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/beego/beego/v2/core/logs"
	"github.com/eclipse/paho.mqtt.golang/packets"
)

type (
	// Broker is MQTT server, will manage client, topic, session, etc.
	Broker struct {
		sync.RWMutex
		egName string
		name   string
		spec   *Spec

		listener net.Listener
		clients  map[string]*Client
		tlsCfg   *tls.Config

		sessMgr  *SessionManager
		topicMgr *TopicManager

		// done is the channel for shutdowning this proxy.
		done      chan struct{}
		closeFlag int32
	}

	// HTTPJsonData is json data received from http endpoint used to send back to clients
	HTTPJsonData struct {
		Topic       string `json:"topic"`
		QoS         int    `json:"qos"`
		Payload     string `json:"payload"`
		Base64      bool   `json:"base64"`
		Distributed bool   `json:"distributed"`
	}

	// HTTPSessions is json data used for session related operations, like get all sessions and delete some sessions
	HTTPSessions struct {
		Sessions []*HTTPSession `json:"sessions"`
	}

	// HTTPSession is json data used for session related operations, like get all sessions and delete some sessions
	HTTPSession struct {
		SessionID string `json:"sessionID"`
		Topic     string `json:"topic"`
	}
)

func NewBroker(spec *Spec) *Broker {
	broker := &Broker{
		egName:  spec.EGName,
		name:    spec.Name,
		spec:    spec,
		clients: make(map[string]*Client),
		done:    make(chan struct{}),
	}

	err := broker.setListener()
	if err != nil {
		logs.Error("mqtt broker set listener failed: %v", err)
		return nil
	}

	if spec.TopicCacheSize <= 0 {
		spec.TopicCacheSize = 100000
	}
	broker.topicMgr = newTopicManager(spec.TopicCacheSize)
	broker.sessMgr = newSessionManager(broker)
	go broker.run()
	// ch, closeFunc, err := broker.sessMgr.store.watchDelete(sessionStoreKey(""))
	// if err != nil {
	// 	logs.Error("get watcher for session failed, %v", err)
	// }
	// if ch != nil {
	// 	go broker.watchDelete(ch, func(){})
	// }
	return broker
}

func (b *Broker) setListener() error {
	var l net.Listener
	var err error
	var cfg *tls.Config
	addr := fmt.Sprintf(":%d", b.spec.Port)
	if b.spec.UseTLS {
		cfg, err = b.spec.tlsConfig()
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

func (b *Broker) reconnectWatcher() {
	if b.closed() {
		return
	}

	// check event during reconnect
	clients := []*Client{}
	b.Lock()
	for sessionID, client := range b.clients {
		if ok := b.sessMgr.get(sessionID); ok != nil {
			clients = append(clients, client)
		}
	}
	b.Unlock()

	for _, c := range clients {
		c.close()
	}
}

func (b *Broker) watchDelete(ch <-chan map[string]*string, closeFunc func()) {
	defer closeFunc()
	for {
		select {
		case <-b.done:
			return
		case m := <-ch:
			if m == nil {
				go b.reconnectWatcher()
				return
			}
			for k, v := range m {
				if v != nil {
					continue
				}
				clientID := strings.TrimPrefix(k, sessionStoreKey(""))
				logs.Debug("client %v recv delete watch %v", clientID, v)
				go b.deleteSession(clientID)
			}
		}
	}
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

	client := newClient(connect, b, conn, b.spec.ClientPublishLimit)
	// check auth
	authFail := false

	if authFail {
		connack.ReturnCode = packets.ErrRefusedNotAuthorised
		err := connack.Write(conn)
		if err != nil {
			logs.Error("connack back to client %s failed: %s", connect.ClientIdentifier, err)
		}
		logs.Error("invalid connection %v, client %s auth failed", connack.ReturnCode, connect.ClientIdentifier)
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

	} else if b.spec.MaxAllowedConnection > 0 {
		if len(b.clients) >= b.spec.MaxAllowedConnection {
			logs.Debug("client %v not get connect permission from rate limiter", connect.ClientIdentifier)
			connack.ReturnCode = packets.ErrRefusedServerUnavailable
			err = connack.Write(conn)
			if err != nil {
				logs.Error("connack back to client %s failed: %s", connect.ClientIdentifier, err)
			}
			b.Unlock()
			return
		}
	}
	b.clients[client.info.cid] = client
	b.setSession(client, connect)
	b.Unlock()

	err = connack.Write(conn)
	if err != nil {
		logs.Error("send connack to client %s failed: %s", connect.ClientIdentifier, err)
		return
	}

	client.session.updateEGName(b.egName, b.name)
	topics, qoss, _ := client.session.allSubscribes()
	if len(topics) > 0 {
		err = b.topicMgr.subscribe(topics, qoss, client.info.cid)
		if err != nil {
			logs.Error("client %v use previous session topics %v to subscribe failed: %v", client.info.cid, topics, err)
		}
	}
	go client.writeLoop()
	client.readLoop()
}

func (b *Broker) setSession(client *Client, connect *packets.ConnectPacket) {
	// when clean session is false, previous session exist and previous session not clean session,
	// then we use previous session, otherwise use new session
	prevSess := b.sessMgr.get(connect.ClientIdentifier)
	if !connect.CleanSession && (prevSess != nil) && !prevSess.cleanSession() {
		client.session = prevSess
	} else {
		if prevSess != nil {
			prevSess.close()
		}
		client.session = b.sessMgr.newSessionFromConn(connect)
	}
}

func (b *Broker) sendMsgToClient(topic string, payload []byte, qos byte) {
	subscribers, _ := b.topicMgr.findSubscribers(topic)
	logs.Debug("eg %v send topic %v to client %v", b.egName, topic, subscribers)
	if subscribers == nil {
		logs.Error("eg %v not find subscribers for topic %s", b.egName, topic)
		return
	}

	for clientID, subQoS := range subscribers {
		if subQoS < qos {
			return
		}
		client := b.getClient(clientID)
		if client == nil {
			logs.Debug("client %v not on broker %v in eg %v", clientID, b.name, b.egName)
		} else {
			client.session.publish(topic, payload, qos)
		}
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
func (b *Broker) mqttAPIPrefix(path string) string {
	return fmt.Sprintf(path, b.name)
}

func (b *Broker) currentClients() map[string]struct{} {
	ans := make(map[string]struct{})
	b.Lock()
	for k := range b.clients {
		ans[k] = struct{}{}
	}
	b.Unlock()
	return ans
}

func (b *Broker) setClose() {
	atomic.StoreInt32(&b.closeFlag, 1)
}

func (b *Broker) closed() bool {
	flag := atomic.LoadInt32(&b.closeFlag)
	return flag == 1
}

func (b *Broker) close() {
	b.setClose()
	close(b.done)
	b.listener.Close()
	b.sessMgr.close()

	b.Lock()
	defer b.Unlock()
	for _, v := range b.clients {
		go v.closeAndDelSession()
	}
	b.clients = nil
}
