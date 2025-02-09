package coapserver

import (
	"go-iot/pkg/core"
	"io"

	logs "go-iot/pkg/logger"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
)

func newSession(w mux.ResponseWriter, r *mux.Message, productId string) *CoapSession {
	session := &CoapSession{
		w:         w,
		r:         r,
		productId: productId,
	}
	return session
}

type CoapSession struct {
	w         mux.ResponseWriter
	r         *mux.Message
	productId string
	deviceId  string
}

func (s *CoapSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *CoapSession) GetDeviceId() string {
	return s.deviceId
}

func (s *CoapSession) GetInfo() map[string]any {
	return map[string]any{}
}

func (s *CoapSession) Disconnect() error {
	core.DelSession(s.deviceId)
	return nil
}

func (s *CoapSession) Close() error {
	return nil
}

// 响应普通文本
func (s *CoapSession) Response(msg string) error {
	err := sendResponse(s.w, codes.Content, message.TextPlain, msg)
	if err != nil {
		logs.Warnf("http Response error: %v", err)
	}
	return err
}

// 响应json
func (s *CoapSession) ResponseJSON(msg string) error {
	err := sendResponse(s.w, codes.Content, message.AppJSON, msg)
	if err != nil {
		logs.Warnf("http ResponseJSON error: %v", err)
	}
	return err
}

func (s *CoapSession) readData() error {
	sc := core.GetCodec(s.productId)
	message := s.getBody(s.r) // 10 MB is a lot of text.
	path, _ := s.r.Path()
	sc.OnMessage(&coapContext{
		BaseContext: core.BaseContext{
			DeviceId:  s.GetDeviceId(),
			ProductId: s.productId,
			Session:   s,
		},
		Data: message,
		r:    s.r,
		url:  path,
	})
	return nil
}

func (s *CoapSession) getBody(r *mux.Message) []byte {
	var requestbody []byte
	requestbody, _ = io.ReadAll(r.Body())

	return requestbody
}
