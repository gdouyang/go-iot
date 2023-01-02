package tcpclient

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/codec"
	"go-iot/network/servers"
	tcpserver "go-iot/network/servers/tcp"
	"strconv"
)

type (

	// Spec describes the TcpServer
	TcpClientSpec struct {
		Name                 string                 `json:"name"`
		Host                 string                 `json:"host"`
		Port                 int32                  `json:"port"`
		UseTLS               bool                   `json:"useTLS"`
		Certificate          []servers.Certificate  `json:"certificate"`
		MaxAllowedConnection int                    `json:"maxAllowedConnection"`
		Delimeter            tcpserver.TcpDelimeter `json:"delimeter"`
	}
)

func (spec *TcpClientSpec) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), spec)
	if err != nil {
		return fmt.Errorf("tcpclient spec error:%v", err)
	}
	return nil
}

func (spec *TcpClientSpec) TlsConfig() (*tls.Config, error) {
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

func (spec *TcpClientSpec) SetByConfig(devoper *codec.Device) error {
	spec.Host = devoper.GetConfig("host")
	port, err := strconv.Atoi(devoper.GetConfig("port"))
	if err != nil {
		return errors.New("port is not number")
	}
	spec.Port = int32(port)
	return nil
}
