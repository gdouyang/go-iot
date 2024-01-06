// http服务
package httpserver

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-iot/pkg/network"
)

type (
	// Spec describes the http Server
	HttpServerSpec struct {
		Name        string                `json:"name"`
		Host        string                `json:"host"`
		Port        int32                 `json:"port"`
		UseTLS      bool                  `json:"useTLS"`
		Certificate []network.Certificate `json:"certificate"`
		Routers     []Router              `json:"routers"`
	}
	Router struct {
		Url string `json:"url"`
	}
)

func (spec *HttpServerSpec) FromJson(str string) error {
	if len(str) > 0 {
		err := json.Unmarshal([]byte(str), spec)
		if err != nil {
			return fmt.Errorf("http server spec error: %v", err)
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

func (spec *HttpServerSpec) FromNetwork(network network.NetworkConf) error {
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

func (spec *HttpServerSpec) SetCertificate(conf network.NetworkConf) error {
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
