package router

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/gogf/gf/util/guid"
	"github.com/pingostack/neon/pkg/eventemitter"
	"github.com/sirupsen/logrus"
)

const (
	defaultEventEmitterSize = 100
)

type PeerMeta struct {
	RemoteAddr     string
	LocalAddr      string
	PeerID         string
	RouterID       string
	Domain         string
	URI            string // URI is the path of the request, e.g. /live/room1
	Args           map[string]string
	Producer       bool
	HasAudio       bool
	HasVideo       bool
	HasDataChannel bool
}

type Session interface {
	eventemitter.EventEmitter
	ID() string
	RouterID() string
	Close()
	Set(key, value interface{})
	Get(key interface{}) interface{}
	Logger() *logrus.Entry
	Context() context.Context
	GetNamespace() *Namespace
	SetNamespace(ns *Namespace)
	PeerMeta() PeerMeta
	Finalize(e error)
}

type SessionImpl struct {
	eventemitter.EventEmitter
	id     string
	ns     *Namespace
	kv     *sync.Map
	logger *logrus.Entry
	ctx    context.Context
	cancel context.CancelFunc
	pm     PeerMeta
}

func NewSession(ctx context.Context, pm PeerMeta, logger *logrus.Entry) *SessionImpl {
	s := &SessionImpl{
		id: guid.S(),
		kv: &sync.Map{},
	}

	s.logger = logger.WithFields(logrus.Fields{
		"session": s.id,
		"peer":    pm.PeerID,
	})
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.EventEmitter = eventemitter.NewEventEmitter(s.ctx,
		defaultEventEmitterSize,
		logger.WithField("submodule", "eventemitter"))

	return s
}

func (s *SessionImpl) ID() string {
	return s.id
}

func (s *SessionImpl) RouterID() string {
	return s.pm.RouterID
}

func (s *SessionImpl) Finalize(e error) {
	s.cancel()
	s.logger.WithError(e).Infof("session finalized")
}

func (s *SessionImpl) Set(key, value interface{}) {
	s.kv.Store(key, value)
}

func (s *SessionImpl) Get(key interface{}) interface{} {
	v, _ := s.kv.Load(key)
	return v
}

func (s *SessionImpl) SetNamespace(ns *Namespace) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.ns)), unsafe.Pointer(ns))
}

func (s *SessionImpl) GetNamespace() *Namespace {
	val := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.ns)))
	return (*Namespace)(val)
}

func (s *SessionImpl) Logger() *logrus.Entry {
	return s.logger
}

func (s *SessionImpl) Context() context.Context {
	return s.ctx
}

func (s *SessionImpl) PeerMeta() PeerMeta {
	return s.pm
}
