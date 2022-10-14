package httpserver

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go-iot/network/servers"
	"log"
)

type (
	// Spec describes the http Server
	HttpServerSpec struct {
		Name        string                `json:"name"`
		Host        string                `json:"host"`
		Port        int32                 `json:"port"`
		UseTLS      bool                  `json:"useTLS"`
		Certificate []servers.Certificate `json:"certificate"`
		Paths       []string              `json:"paths"`
	}
)

func (spec *HttpServerSpec) FromJson(str string) {
	err := json.Unmarshal([]byte(str), spec)
	if err != nil {
		log.Panicln(err)
	}
}

func (spec *HttpServerSpec) TlsConfig() (*tls.Config, error) {
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