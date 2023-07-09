package tcpclient

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	tcpserver "go-iot/pkg/network/servers/tcp"
	"strconv"
)

func init() {
	network.RegNetworkMetaConfigCreator(string(network.TCP_CLIENT), func() core.CodecMetaConfig {
		list := []core.MetaConfig{
			{Property: "host", Type: "string", Buildin: true, Desc: "The host of remote [eg: 127.0.0.1]"},
			{Property: "port", Type: "number", Buildin: true, Desc: "The port of remote"},
		}
		return core.CodecMetaConfig{MetaConfigs: list}
	})
}

type (

	// Spec describes the TcpServer
	TcpClientSpec struct {
		Name                 string                 `json:"name"`
		Host                 string                 `json:"host"`
		Port                 int32                  `json:"port"`
		UseTLS               bool                   `json:"useTLS"`
		Certificate          []network.Certificate  `json:"certificate"`
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

func (spec *TcpClientSpec) FromNetwork(network network.NetworkConf) error {
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

func (spec *TcpClientSpec) SetByConfig(devoper *core.Device) error {
	spec.Host = devoper.GetConfig("host")
	port, err := strconv.Atoi(devoper.GetConfig("port"))
	if err != nil {
		return errors.New("port is not number")
	}
	spec.Port = int32(port)
	return nil
}

func (spec *TcpClientSpec) SetCertificate(conf network.NetworkConf) error {
	if len(conf.CertBase64) == 0 || len(conf.KeyBase64) == 0 {
		return nil
	}
	cert, err := base64.StdEncoding.DecodeString(conf.CertBase64)
	if err != nil {
		return fmt.Errorf("tcp server cert error: %v", err)
	}
	key, err := base64.StdEncoding.DecodeString(conf.KeyBase64)
	if err != nil {
		return fmt.Errorf("tcp server key error: %v", err)
	}
	spec.Certificate = []network.Certificate{{Key: string(key), Cert: string(cert)}}
	return nil
}
