package tcpserver

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go-iot/models/network"
)

type (
	// Spec describes the TcpServer
	TcpServerSpec struct {
		Name                 string                `json:"name"`
		Host                 string                `json:"host"`
		Port                 uint16                `json:"port"`
		UseTLS               bool                  `json:"useTLS"`
		Certificate          []network.Certificate `json:"certificate"`
		MaxAllowedConnection int                   `json:"maxAllowedConnection"`
	}
)

func (spec *TcpServerSpec) FromJson(str string) {
	json.Unmarshal([]byte(str), spec)
}

func (spec *TcpServerSpec) TlsConfig() (*tls.Config, error) {
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
