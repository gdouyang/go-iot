package modbus

type session struct {
	deviceId  string
	productId string
}

func (s *session) Disconnect() error {
	return nil
}

func (s *session) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *session) GetDeviceId() string {
	return s.deviceId
}
