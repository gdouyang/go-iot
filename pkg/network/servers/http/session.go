package httpserver

import (
	"compress/gzip"
	"go-iot/pkg/core"
	"io"
	"net/http"

	logs "go-iot/pkg/logger"
)

func newSession(w http.ResponseWriter, r *http.Request, productId string) *HttpSession {
	session := &HttpSession{
		w:         w,
		r:         r,
		productId: productId,
	}
	return session
}

type HttpSession struct {
	w         http.ResponseWriter
	r         *http.Request
	productId string
	deviceId  string
}

func (s *HttpSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *HttpSession) GetDeviceId() string {
	return s.deviceId
}

func (s *HttpSession) Disconnect() error {
	_, err := s.w.Write([]byte(""))
	if err != nil {
		logs.Warnf("http Disconnect error: %v", err)
	}
	return err
}

func (s *HttpSession) Close() error {
	return s.Disconnect()
}

func (s *HttpSession) Response(msg string) error {
	_, err := s.w.Write([]byte(msg))
	if err != nil {
		logs.Warnf("http Response error: %v", err)
	}
	return err
}

func (s *HttpSession) ResponseJSON(msg string) error {
	s.ResponseHeader("Content-Type", "application/json; charset=utf-8")
	_, err := s.w.Write([]byte(msg))
	if err != nil {
		logs.Warnf("http ResponseJSON error: %v", err)
	}
	return err
}

func (s *HttpSession) ResponseHeader(key string, value string) {
	s.w.Header().Add("Content-Type", "application/json; charset=utf-8")
}

func (s *HttpSession) readData() error {
	sc := core.GetCodec(s.productId)
	message := s.getBody(s.r, 1024)
	sc.OnMessage(&httpContext{
		BaseContext: core.BaseContext{
			DeviceId:  s.GetDeviceId(),
			ProductId: s.productId,
			Session:   s,
		},
		Data: message,
		r:    s.r,
	})
	return nil
}

func (s *HttpSession) getBody(r *http.Request, MaxMemory int64) []byte {
	if r.Body == nil {
		return []byte{}
	}

	var requestbody []byte
	safe := &io.LimitedReader{R: r.Body, N: MaxMemory}
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(safe)
		if err != nil {
			return nil
		}
		requestbody, _ = io.ReadAll(reader)
	} else {
		requestbody, _ = io.ReadAll(safe)
	}

	r.Body.Close()
	return requestbody
}
