package mqttserver

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go-iot/network/servers"

	"github.com/beego/beego/v2/core/logs"
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

type (
	// Spec describes the MQTTProxy.
	MQTTServerSpec struct {
		Host                 string                `json:"host"`
		Name                 string                `json:"name"`
		Port                 int32                 `json:"port"`
		UseTLS               bool                  `json:"useTLS"`
		Certificate          []servers.Certificate `json:"certificate"`
		MaxAllowedConnection int                   `json:"maxAllowedConnection"`
	}
)

func (spec *MQTTServerSpec) FromJson(str string) {
	err := json.Unmarshal([]byte(str), spec)
	if err != nil {
		logs.Error(err)
	}
}

func (spec *MQTTServerSpec) TlsConfig() (*tls.Config, error) {
	var certificates []tls.Certificate

	for _, c := range spec.Certificate {
		cert, err := tls.X509KeyPair([]byte(c.Cert), []byte(c.Key))
		if err != nil {
			return nil, fmt.Errorf("generate x509 key pair for %s failed: %s ", c.Name, err)
		}
		certificates = append(certificates, cert)
	}
	if len(certificates) == 0 {
		return nil, fmt.Errorf("none valid certs and secret")
	}

	return &tls.Config{Certificates: certificates}, nil
}
