package router

import (
	"context"
	"sync"
	"time"

	"github.com/gogf/gf/os/gtimer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Router interface {
	ID() string
	AddSession(session Session) error
	Namespace() *Namespace
	Context() context.Context
	Closed() bool
}

type RouterImpl struct {
	ctx         context.Context
	cancel      context.CancelFunc
	id          string
	ns          *Namespace
	producer    Session
	subscribers map[string]Session
	lock        sync.RWMutex
	logger      *logrus.Entry
	closed      bool
	params      RouterParams
	closeTimer  *gtimer.Entry
	stream      Stream
}

func NewRouter(ctx context.Context, ns *Namespace, params RouterParams, id string, logger *logrus.Entry) Router {
	r := &RouterImpl{
		ns:          ns,
		params:      params,
		id:          id,
		subscribers: make(map[string]Session),
		logger:      logger.WithField("router", id),
		stream:      NewStreamImpl(ctx, id),
	}

	r.ctx, r.cancel = context.WithCancel(ctx)

	r.logger.Infof("router created")

	return r
}

func (r *RouterImpl) ID() string {
	return r.id
}

func (r *RouterImpl) addProducer(session Session) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		return ErrRouterClosed
	}

	if r.closeTimer != nil {
		r.closeTimer.Stop()
		r.closeTimer.Close()
		r.closeTimer = nil
	}

	if r.producer != nil {
		r.producer.Finalize(ErrProducerRepeated)
	}

	r.producer = session

	if err := r.stream.AddFrameSource(session.FrameSource()); err != nil {
		r.logger.WithError(err).Error("failed to add frame source")
		return errors.Wrap(err, "failed to add frame source")
	}

	go r.waitSessionDone(session)

	return nil
}

func (r *RouterImpl) addSubscriber(session Session) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		return ErrRouterClosed
	}

	if _, ok := r.subscribers[session.ID()]; ok {
		return ErrSessionAlreadyExists
	}

	r.subscribers[session.ID()] = session

	if err := r.stream.AddFrameDestination(session.FrameDestination()); err != nil {
		r.logger.WithError(err).Error("failed to add frame destination")
		return errors.Wrap(err, "failed to add frame destination")
	}

	go r.waitSessionDone(session)

	return nil
}

func (r *RouterImpl) AddSession(session Session) error {
	if session.PeerParams().Producer {
		return r.addProducer(session)
	} else {
		return r.addSubscriber(session)
	}
}

func (r *RouterImpl) waitSessionDone(s Session) {
	<-s.Context().Done()

	close := func(e error) {
		if r.closed {
			return
		}

		r.closed = true
		r.cancel()

		if r.producer != nil {
			r.producer.Finalize(e)
		}

		for _, s := range r.subscribers {
			s.Finalize(e)
		}

		r.logger.Infof("router closed")
	}

	delayClose := func() {
		r.closeTimer = gtimer.AddOnce(time.Duration(r.params.IdleSubscriberTimeout)*time.Second, func() {
			r.lock.Lock()
			defer r.lock.Unlock()

			if r.producer != nil {
				r.logger.Infof("router idle timeout")
				close(ErrSessionIdleTimeout)
			}
		})
		r.closeTimer.Start()
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if s.PeerParams().Producer {
		r.producer = nil
		r.logger.Infof("producer %s removed", s.ID())
		if len(r.subscribers) > 0 {
			if r.params.IdleSubscriberTimeout > 0 {
				delayClose()
				return
			} else if r.params.IdleSubscriberTimeout == 0 {
				r.logger.Infof("router idle timeout is 0, close router")
				close(ErrProducerEmpty)
				return
			} else {
				r.logger.Debugf("router idle timeout disabled, keep router")
			}
		}
	} else {
		delete(r.subscribers, s.ID())
		r.logger.Infof("subscriber %s removed", s.ID())
	}

	if len(r.subscribers) == 0 && r.producer == nil {
		r.logger.Infof("no producer and subscribers, close router")
		close(nil)
	}
}

func (r *RouterImpl) Namespace() *Namespace {
	return r.ns
}

func (r *RouterImpl) Context() context.Context {
	return r.ctx
}

func (r *RouterImpl) Closed() bool {
	return r.closed
}
