package core

import (
	"sync"
)

type Stream struct {
	id       string
	sessions map[string]Session
	lock     sync.RWMutex
}

func NewStream(id string) *Stream {
	return &Stream{
		id:       id,
		sessions: make(map[string]Session),
	}
}

func (s *Stream) ID() string {
	return s.id
}

func (s *Stream) AddSessionIfNotExists(session Session) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.sessions[session.ID()]; !ok {
		s.sessions[session.ID()] = session
		return true
	}

	return false
}

func (s *Stream) DeleteSession(session Session) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.sessions, session.ID())
}
