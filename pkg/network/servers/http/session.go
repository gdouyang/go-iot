package httpserver

import (
	"compress/gzip"
	"go-iot/pkg/core"
	"io"
	"net/http"

	logs "go-iot/pkg/logger"
)

func newSession(w http.ResponseWriter, r *http.Request, productId string) *httpSession {
	session := &httpSession{
		w:         w,
		r:         r,
		productId: productId,
	}
	return session
}

type httpSession struct {
	w         http.ResponseWriter
	r         *http.Request
	productId string
	deviceId  string
}

func (s *httpSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *httpSession) GetDeviceId() string {
	return s.deviceId
}

func (s *httpSession) Disconnect() error {
	_, err := s.w.Write([]byte(""))
	if err != nil {
		logs.Warnf("http Disconnect error: %v", err)
	}
	return err
}

func (s *httpSession) Response(msg string) error {
	_, err := s.w.Write([]byte(msg))
	if err != nil {
		logs.Warnf("http Response error: %v", err)
	}
	return err
}

func (s *httpSession) ResponseJSON(msg string) error {
	s.ResponseHeader("Content-Type", "application/json; charset=utf-8")
	_, err := s.w.Write([]byte(msg))
	if err != nil {
		logs.Warnf("http ResponseJSON error: %v", err)
	}
	return err
}

func (s *httpSession) ResponseHeader(key string, value string) {
	s.w.Header().Add("Content-Type", "application/json; charset=utf-8")
}

func (s *httpSession) readData() error {
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

func (s *httpSession) getBody(r *http.Request, MaxMemory int64) []byte {
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
