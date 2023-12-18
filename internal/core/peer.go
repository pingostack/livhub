package core

type PeerReq struct {
	Session  Session
	Domain   string
	StreamID string
	Args     map[string]string
}

type PublishReq struct {
	PeerReq
}

type PlayReq struct {
	PeerReq
}

type Session interface {
	ID() string
	Close()
}

type SessionImpl struct {
	id string
}

func NewSession(id string) *SessionImpl {
	return &SessionImpl{
		id: id,
	}
}

func (s *SessionImpl) ID() string {
	return s.id
}

func (s *SessionImpl) Close() {
}
