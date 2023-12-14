package pms

type Session struct {
	sessionId string
}

func NewSession(sessionId string) *Session {
	return &Session{
		sessionId: sessionId,
	}
}

func (s *Session) SessionId() string {
	return s.sessionId
}
