package tcpserver

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go-iot/network/servers"
	"log"
)

type (

	// Spec describes the TcpServer
	TcpServerSpec struct {
		Name                 string                `json:"name"`
		Host                 string                `json:"host"`
		Port                 uint16                `json:"port"`
		UseTLS               bool                  `json:"useTLS"`
		Certificate          []servers.Certificate `json:"certificate"`
		MaxAllowedConnection int                   `json:"maxAllowedConnection"`
		Delimeter            TcpDelimeter          `json:"delimeter"`
	}
	TcpDelimeter struct {
		Type      DelimType `json:"type"`      // Delimited(分隔符) FixLength(固定长度)
		Delimited string    `json:"delimited"` // 分隔符
		Length    int32     `json:"length"`    // 长度
		SplitFunc string    `json:"splitFunc"`
	}
)

func (spec *TcpServerSpec) FromJson(str string) {
	err := json.Unmarshal([]byte(str), spec)
	if err != nil {
		log.Panicln(err)
	}
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
