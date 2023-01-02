package mqttclient

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/codec"
	"go-iot/network/servers"
	"strconv"
)

type MQTTClientSpec struct {
	Host         string                `json:"host"`
	Port         int32                 `json:"port"`
	ClientId     string                `json:"clientId"`
	Username     string                `json:"username"`
	Password     string                `json:"password"`
	Topics       map[string]int        `json:"topics"`
	CleanSession bool                  `json:"cleanSession"`
	UseTLS       bool                  `json:"useTLS"`
	Certificate  []servers.Certificate `json:"certificate"`
}

func (spec *MQTTClientSpec) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), spec)
	if err != nil {
		return fmt.Errorf("mqtt client spec error: %v", err)
	}
	return nil
}

func (spec *MQTTClientSpec) TlsConfig() (*tls.Config, error) {
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

func (spec *MQTTClientSpec) SetByConfig(devoper *codec.Device) error {
	spec.Host = devoper.GetConfig("host")
	port, err := strconv.Atoi(devoper.GetConfig("port"))
	if err != nil {
		return errors.New("port is not number")
	}
	spec.Port = int32(port)
	spec.ClientId = devoper.GetConfig("clientId")
	spec.Username = devoper.GetConfig("username")
	spec.Password = devoper.GetConfig("password")
	return nil
}
