package router

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

type PeerInfo struct {
	Session Session
	Domain  string
	URI     string // URI is the path of the request, e.g. /live/room1
	Args    map[string]string
}

type PublishReq struct {
	Ctx context.Context
	PeerInfo
}

type PlayReq struct {
	Ctx context.Context
	PeerInfo
}

type Session interface {
	ID() string
	Close()
	Set(key, value interface{})
	Get(key interface{}) interface{}
	SetRouter(router *Router)
	GetRouter() *Router
	Logger() *logrus.Entry
}

type SessionImpl struct {
	id     string
	ns     *Namespace
	router *Router
	kv     *sync.Map
	lock   sync.RWMutex
	logger *logrus.Entry
	chIn   chan interface{}
	chOut  chan interface{}
}

func NewSession(id string, logger *logrus.Entry) *SessionImpl {
	return &SessionImpl{
		id:     id,
		kv:     &sync.Map{},
		logger: logger,
		chIn:   make(chan interface{}),
		chOut:  make(chan interface{}),
	}
}

func (s *SessionImpl) ID() string {
	return s.id
}

func (s *SessionImpl) Close() {
}

func (s *SessionImpl) Set(key, value interface{}) {
	s.kv.Store(key, value)
}

func (s *SessionImpl) Get(key interface{}) interface{} {
	v, _ := s.kv.Load(key)
	return v
}

func (s *SessionImpl) SetRouter(router *Router) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.router = router
}

func (s *SessionImpl) GetNamespace() *Namespace {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ns
}

func (s *SessionImpl) GetRouter() *Router {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.router
}

func (s *SessionImpl) Logger() *logrus.Entry {
	return s.logger
}
