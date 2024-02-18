package core

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/gogf/gf/util/guid"
	"github.com/pingostack/neon/internal/core/router"
	"github.com/pingostack/neon/pkg/eventemitter"
	"github.com/pingostack/neon/pkg/streaminterceptor"
	"github.com/sirupsen/logrus"
)

type SessionImpl struct {
	eventemitter.EventEmitter
	id       string
	ns       *router.Namespace
	kv       *sync.Map
	logger   *logrus.Entry
	ctx      context.Context
	cancel   context.CancelFunc
	peerMeta router.PeerMeta
	router   router.Router
}

func NewSession(ctx context.Context, peerMeta router.PeerMeta, logger *logrus.Entry) router.Session {
	session := &SessionImpl{
		id: guid.S(),
		kv: &sync.Map{},
	}

	session.logger = logger.WithFields(logrus.Fields{
		"session": session.id,
		"peer":    peerMeta.PeerID,
	})
	session.ctx, session.cancel = context.WithCancel(ctx)
	session.EventEmitter = eventemitter.NewEventEmitter(session.ctx,
		defaultEventEmitterSize,
		logger.WithField("submodule", "eventemitter"))

	return session
}

func (session *SessionImpl) ID() string {
	return session.id
}

func (session *SessionImpl) RouterID() string {
	return session.peerMeta.RouterID
}

func (session *SessionImpl) Finalize(e error) {
	session.cancel()
	session.logger.WithError(e).Infof("session finalized")
}

func (session *SessionImpl) Set(key, value interface{}) {
	session.kv.Store(key, value)
}

func (session *SessionImpl) Get(key interface{}) interface{} {
	v, _ := session.kv.Load(key)
	return v
}

func (session *SessionImpl) SetNamespace(ns *router.Namespace) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&session.ns)), unsafe.Pointer(ns))
}

func (session *SessionImpl) GetNamespace() *router.Namespace {
	val := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&session.ns)))
	return (*router.Namespace)(val)
}

func (session *SessionImpl) Logger() *logrus.Entry {
	return session.logger
}

func (session *SessionImpl) Context() context.Context {
	return session.ctx
}

func (session *SessionImpl) PeerMeta() router.PeerMeta {
	return session.peerMeta
}

func (session *SessionImpl) Join() error {
	return defaultServ.Join(session.ctx, session)
}

func (session *SessionImpl) GetRouter() router.Router {
	return session.router
}

func (session *SessionImpl) SetRouter(r router.Router) {
	session.router = r
}

func (session *SessionImpl) Read([]byte, streaminterceptor.Attributes) (int, streaminterceptor.Attributes, error) {
	return 0, nil, nil
}
