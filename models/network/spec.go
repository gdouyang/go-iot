package network

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
)

type (
	// Spec describes the MQTTProxy.
	MQTTProxySpec struct {
		EGName               string        `json:"egName"`
		Name                 string        `json:"name"`
		Port                 uint16        `json:"port"`
		UseTLS               bool          `json:"useTLS"`
		Certificate          []Certificate `json:"certificate"`
		MaxAllowedConnection int           `json:"maxAllowedConnection"`
	}

	// Certificate describes TLS certifications.
	Certificate struct {
		Name string `json:"name"`
		Cert string `json:"cert"`
		Key  string `json:"key"`
	}
)

func (spec *MQTTProxySpec) FromJson(str string) {
	json.Unmarshal([]byte(str), spec)
}

func (spec *MQTTProxySpec) TlsConfig() (*tls.Config, error) {
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
