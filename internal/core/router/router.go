package router

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gogf/gf/os/gtimer"
	"github.com/pingostack/neon/internal/err"
	"github.com/sirupsen/logrus"
)

type Router struct {
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
}

func NewRouter(ctx context.Context, ns *Namespace, params RouterParams, id string, logger *logrus.Entry) *Router {
	r := &Router{
		ns:          ns,
		params:      params,
		id:          id,
		subscribers: make(map[string]Session),
		logger:      logger.WithField("router", id),
	}

	r.ctx, r.cancel = context.WithCancel(ctx)

	r.logger.Infof("router created")

	return r
}

func (r *Router) ID() string {
	return r.id
}

func (r *Router) SetProducer(session Session) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		return errors.Join(fmt.Errorf("router %s closed", r.id), err.ErrRouterClosed)
	}

	if r.producer != nil {
		r.producer.Finalize(err.ErrProducerRepeated)
	}

	r.producer = session

	go r.waitSession(true, session)

	return nil
}

func (r *Router) AddSubscriber(session Session) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		return errors.Join(fmt.Errorf("router %s closed", r.id), err.ErrRouterClosed)
	}

	if _, ok := r.subscribers[session.ID()]; ok {
		return errors.Join(fmt.Errorf("session %s already exists", session.ID()), err.ErrSessionAlreadyExists)
	}

	r.subscribers[session.ID()] = session

	go r.waitSession(false, session)

	return nil
}

func (r *Router) waitSession(isProducer bool, s Session) {
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
		gtimer.AddOnce(time.Duration(r.params.IdleSubscriberTimeout)*time.Second, func() {
			r.lock.Lock()
			defer r.lock.Unlock()

			if r.producer != nil {
				r.logger.Infof("router idle timeout")
				close(err.ErrSessionIdleTimeout)
			}
		})
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if isProducer {
		r.producer = nil
		r.logger.Infof("producer %s removed", s.ID())
		if len(r.subscribers) > 0 {
			if r.params.IdleSubscriberTimeout > 0 {
				delayClose()
				return
			} else if r.params.IdleSubscriberTimeout == 0 {
				r.logger.Infof("router idle timeout is 0, close router")
				close(err.ErrProducerEmpty)
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

func (r *Router) Namespace() *Namespace {
	return r.ns
}

func (r *Router) Context() context.Context {
	return r.ctx
}
