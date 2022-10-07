package httpserver

import (
	"encoding/json"
	"go-iot/codec"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
)

func newSession(w http.ResponseWriter, r *http.Request) codec.Session {
	session := &httpSession{w: w, r: r}
	return session
}

type httpSession struct {
	w        http.ResponseWriter
	r        *http.Request
	deviceId string
}

func (s *httpSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *httpSession) GetDeviceId() string {
	return s.deviceId
}

func (s *httpSession) Send(msg interface{}) error {
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

func (s *httpSession) Disconnect() error {
	_, err := s.w.Write([]byte(""))
	return err
}
