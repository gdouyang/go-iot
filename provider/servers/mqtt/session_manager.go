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
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"github.com/eclipse/paho.mqtt.golang/packets"
)

type (
	// SessionManager manage the status of session for clients
	SessionManager struct {
		broker     *Broker
		sessionMap sync.Map
		storeCh    chan SessionStore
		done       chan struct{}
	}

	// SessionStore for session store, key is session clientID, value is session yaml marshal value
	SessionStore struct {
		key   string
		value string
	}
)

func newSessionManager(b *Broker) *SessionManager {
	sm := &SessionManager{
		broker:  b,
		storeCh: make(chan SessionStore),
		done:    make(chan struct{}),
	}
	go sm.doStore()
	return sm
}

func (sm *SessionManager) close() {
	close(sm.done)
}

func (sm *SessionManager) doStore() {
	for {
		select {
		case <-sm.done:
			return
		case kv := <-sm.storeCh:
			logs.Debug("session manager store session %v", kv.key)
			sess := &Session{}
			sess.broker = sm.broker
			sess.storeCh = sm.storeCh
			sess.done = make(chan struct{})
			sess.pending = make(map[uint16]*Message)
			sess.pendingQueue = []uint16{}

			sess.info = &SessionInfo{}
			err := sess.decode(kv.value)
			if err == nil {
				sm.sessionMap.Store(sess.info.ClientID, sess)
			}
		}
	}
}

func (sm *SessionManager) newSessionFromConn(connect *packets.ConnectPacket) *Session {
	s := &Session{}
	s.init(sm, sm.broker, connect)
	sm.sessionMap.Store(connect.ClientIdentifier, s)
	go s.backgroundResendPending()
	return s
}

func (sm *SessionManager) get(clientID string) *Session {
	if val, ok := sm.sessionMap.Load(clientID); ok {
		return val.(*Session)
	}

	return nil
}

func (sm *SessionManager) delLocal(clientID string) {
	if val, ok := sm.sessionMap.LoadAndDelete(clientID); ok {
		sess := val.(*Session)
		sess.close()
	}
}