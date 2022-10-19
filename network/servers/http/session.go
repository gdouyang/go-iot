package httpserver

import (
	"compress/gzip"
	"encoding/json"
	"go-iot/codec"
	"io"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
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
	return err
}

func (s *httpSession) Response(msg interface{}) error {
	var err error
	switch t := msg.(type) {
	case string:
		_, err = s.w.Write([]byte(t))
	case map[string]interface{}:
		b, err1 := json.Marshal(t)
		if err1 != nil {
			logs.Warn("map to json string error:", err)
		}
		s.w.Header().Add("Content-Type", "")
		_, err = s.w.Write([]byte(b))
	default:
		logs.Warn("unsupport msg:", msg)
	}
	if err != nil {
		logs.Warn("Error during message writing:", err)
	}
	return err
}

func (s *httpSession) readData() error {
	sc := codec.GetCodec(s.productId)
	message := s.getBody(s.r, 1024)
	sc.OnMessage(&httpContext{
		BaseContext: codec.BaseContext{
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
