package websocketsocker

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network/servers"
)

type (
	// Spec describes the websocket Server
	WebsocketServerSpec struct {
		Name                 string                `json:"name"`
		Host                 string                `json:"host"`
		Port                 int32                 `json:"port"`
		UseTLS               bool                  `json:"useTLS"`
		Certificate          []servers.Certificate `json:"certificate"`
		MaxAllowedConnection int                   `json:"maxAllowedConnection"`
		Routers              []Router              `json:"routers"`
	}
	Router struct {
		Url string `json:"url"`
	}
)

func (spec *WebsocketServerSpec) FromJson(str string) error {
	if len(str) > 0 {
		err := json.Unmarshal([]byte(str), spec)
		if err != nil {
			return fmt.Errorf("websocket server spec error: %v", err)
		}
	}
	routers := []Router{}
	for _, v := range spec.Routers {
		if len(v.Url) > 0 {
			routers = append(routers, v)
		}
	}
	spec.Routers = routers
	return nil
}

func (spec *WebsocketServerSpec) FromNetwork(network core.NetworkConf) error {
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

func (spec *WebsocketServerSpec) TlsConfig() (*tls.Config, error) {
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

func (spec *WebsocketServerSpec) SetCertificate(network core.NetworkConf) error {
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
