package mqttserver

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
)

func init() {
	network.RegNetworkMetaConfigCreator(string(network.MQTT_BROKER), func() core.CodecMetaConfig {
		list := []core.MetaConfig{
			{Property: "username", Type: "string", Buildin: true, Desc: "The username of mqtt"},
			{Property: "password", Type: "password", Buildin: true, Desc: "The password of mqtt"},
		}
		return core.CodecMetaConfig{MetaConfigs: list}
	})
}

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
		Certificate          []network.Certificate `json:"certificate"`
		MaxAllowedConnection int                   `json:"maxAllowedConnection"`
	}
)

func (spec *MQTTServerSpec) FromJson(str string) error {
	if len(str) == 0 {
		return nil
	}
	err := json.Unmarshal([]byte(str), spec)
	if err != nil {
		return fmt.Errorf("mqtt broker spec error: %v", err)
	}
	return nil
}

func (spec *MQTTServerSpec) FromNetwork(network network.NetworkConf) error {
	err := spec.FromJson(network.Configuration)
	if err != nil {
		return err
	}
	if spec.UseTLS {
		err = spec.SetCertificate(network)
		if err != nil {
			return err
		}
	}
	return nil
}

func (spec *MQTTServerSpec) TlsConfig() (*tls.Config, error) {
	var certificates []tls.Certificate

	for _, c := range spec.Certificate {
		certPEMBlock := []byte(c.Cert)
		keyPEMBlock := []byte(c.Key)
		cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
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

func (spec *MQTTServerSpec) SetCertificate(conf network.NetworkConf) error {
	if len(conf.CertBase64) == 0 || len(conf.KeyBase64) == 0 {
		return nil
	}
	cert, err := base64.StdEncoding.DecodeString(conf.CertBase64)
	if err != nil {
		return fmt.Errorf("mqtt server cert error: %v", err)
	}
	key, err := base64.StdEncoding.DecodeString(conf.KeyBase64)
	if err != nil {
		return fmt.Errorf("mqtt server key error: %v", err)
	}
	spec.Certificate = []network.Certificate{{Key: string(key), Cert: string(cert)}}
	return nil
}
