package core

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/gogf/gf/util/guid"
	"github.com/pingostack/livhub/internal/core/router"
	"github.com/pingostack/livhub/pkg/deliver"
	"github.com/sirupsen/logrus"
)

type SessionImpl struct {
	id               string
	ns               *router.Namespace
	kv               *sync.Map
	logger           *logrus.Entry
	ctx              context.Context
	cancel           context.CancelFunc
	params           router.PeerParams
	router           router.Router
	frameSource      deliver.FrameSource
	frameDestination deliver.FrameDestination
	onceClose        sync.Once
}

func NewSession(ctx context.Context, params router.PeerParams, logger *logrus.Entry) router.Session {
	session := &SessionImpl{
		id:     guid.S(),
		kv:     &sync.Map{},
		params: params,
	}

	session.logger = logger.WithFields(logrus.Fields{
		"session": session.id,
		"peer":    params.PeerID,
	})
	session.ctx, session.cancel = context.WithCancel(ctx)

	return session
}

func (session *SessionImpl) ID() string {
	return session.id
}

func (session *SessionImpl) RouterID() string {
	return session.params.RouterID
}

func (session *SessionImpl) close(e error) {
	session.onceClose.Do(func() {
		session.cancel()
		session.logger.WithError(e).Infof("session closed")
	})
}

func (session *SessionImpl) Finalize(e error) {
	session.close(e)
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

func (session *SessionImpl) PeerParams() router.PeerParams {
	return session.params
}

func (session *SessionImpl) Join() error {
	if session.params.Producer && session.frameSource == nil {
		return ErrFrameSourceNil
	}

	if !session.params.Producer && session.frameDestination == nil {
		return ErrFrameDestinationNil
	}

	return defaultServ.Join(session.ctx, session)
}

func (session *SessionImpl) GetRouter() router.Router {
	return session.router
}

func (session *SessionImpl) SetRouter(r router.Router) {
	session.router = r
}

func (session *SessionImpl) BindFrameSource(src deliver.FrameSource) error {
	if session.frameSource != nil {
		return ErrFrameSourceAlreadyBound
	}
	session.frameSource = src
	go func(fs deliver.FrameSource) {
		for {
			select {
			case <-session.ctx.Done():
				if session.frameSource != nil {
					session.logger.Debug("session closed, closing frame source")
					session.frameSource.Close()
				}
				return
			case <-fs.Context().Done():
				if fs == session.frameSource {
					session.logger.Debug("frame source closed, closing session")
					session.close(ErrFrameSourceClosed)
					return
				}
			}
		}
	}(src)

	return nil
}

func (session *SessionImpl) BindFrameDestination(dest deliver.FrameDestination) error {
	if session.frameDestination != nil {
		return ErrFrameDestinationBound
	}

	session.frameDestination = dest
	go func(fd deliver.FrameDestination) {
		for {
			select {
			case <-session.ctx.Done():
				if session.frameDestination != nil {
					session.logger.Debug("session closed, closing frame destination")
					session.frameDestination.Close()
				}
				return
			case <-fd.Context().Done():
				if fd == session.frameDestination {
					session.logger.Debug("frame destination closed, closing session")
					session.close(ErrFrameDestinationClosed)
				}
			}
		}
	}(dest)

	return nil
}

func (session *SessionImpl) FrameSource() deliver.FrameSource {
	return session.frameSource
}

func (session *SessionImpl) FrameDestination() deliver.FrameDestination {
	return session.frameDestination
}
