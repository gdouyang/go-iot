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
	"fmt"
)

const (
	sessionPrefix              = "/mqtt/sessionMgr/clientID/%s"
	topicPrefix                = "/mqtt/topicMgr/topic/%s"
	mqttAPITopicPublishPrefix  = "/mqttproxy/%s/topics/publish"
	mqttAPISessionQueryPrefix  = "/mqttproxy/%s/session/query"
	mqttAPISessionDeletePrefix = "/mqttproxy/%s/sessions"
)

// PacketType is mqtt packet type
type PacketType string

const (
	// Connect is connect type of MQTT packet
	Connect PacketType = "Connect"

	// Disconnect is disconnect type of MQTT packet
	Disconnect PacketType = "Disconnect"

	// Publish is publish type of MQTT packet
	Publish PacketType = "Publish"

	// Subscribe is subscribe type of MQTT packet
	Subscribe PacketType = "Subscribe"

	// Unsubscribe is unsubscribe type of MQTT packet
	Unsubscribe PacketType = "Unsubscribe"
)

var pipelinePacketTypes = map[PacketType]struct{}{
	Connect:     {},
	Disconnect:  {},
	Publish:     {},
	Subscribe:   {},
	Unsubscribe: {},
}

func sessionStoreKey(clientID string) string {
	return fmt.Sprintf(sessionPrefix, clientID)
}