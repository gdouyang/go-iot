package codec

var sessionManager *SessionManager = &SessionManager{}

func GetSessionManager() *SessionManager {
	return sessionManager
}

type SessionManager struct {
	sessionMap map[string]Session
}

func (sm *SessionManager) GetSession(deviceId string) Session {
	s := sm.sessionMap[deviceId]
	return s
}

func (sm *SessionManager) PutSession(deviceId string, session Session) {
	sm.sessionMap[deviceId] = session
}
