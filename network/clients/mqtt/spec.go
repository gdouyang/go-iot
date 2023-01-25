package mqttclient

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/codec"
	"go-iot/network/servers"
	"strconv"
)

func init() {
	codec.RegNetworkMetaConfigCreator(string(codec.MQTT_CLIENT), func() codec.DefaultMetaConfig {
		list := []codec.ProductMetaConfig{
			{Property: "host", Type: "string", Buildin: true, Desc: "The host of mqtt broker [eg: 127.0.0.1]"},
			{Property: "port", Type: "number", Buildin: true, Desc: "The port of mqtt broker"},
			{Property: "clientId", Type: "string", Buildin: true, Desc: "The clientId of mqtt"},
			{Property: "username", Type: "string", Buildin: true, Desc: "The username of mqtt"},
			{Property: "password", Type: "password", Buildin: true, Desc: "The password of mqtt"},
		}
		return codec.DefaultMetaConfig{MetaConfigs: list}
	})
}

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

func (spec *MQTTClientSpec) FromNetwork(network codec.NetworkConf) error {
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

func (spec *MQTTClientSpec) SetCertificate(network codec.NetworkConf) error {
	if len(network.CertBase64) == 0 || len(network.KeyBase64) == 0 {
		return nil
	}
	cert, err := base64.StdEncoding.DecodeString(network.CertBase64)
	if err != nil {
		return fmt.Errorf("tcp server cert error: %v", err)
	}
	key, err := base64.StdEncoding.DecodeString(network.KeyBase64)
	if err != nil {
		return fmt.Errorf("tcp server key error: %v", err)
	}
	spec.Certificate = []servers.Certificate{{Key: string(key), Cert: string(cert)}}
	return nil
}
