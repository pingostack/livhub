package core

import (
	"context"

	"github.com/pingostack/neon/internal/core/middleware"
	"github.com/pingostack/neon/internal/core/router"
	"github.com/pingostack/neon/pkg/eventemitter"
)

type serv struct {
	*router.NSManager
	middleware middleware.Matcher
	ee         eventemitter.EventEmitter
	ctx        context.Context
}

type ServerOption func(*serv)

var (
	defaultServ *serv
)

const (
	defaultEventEmitterSize = 100
)

func NewServ(ctx context.Context, params router.NSManagerParams) *serv {
	s := &serv{
		ctx:        ctx,
		middleware: middleware.New(),
		ee:         eventemitter.NewEventEmitter(ctx, defaultEventEmitterSize, DefaultLogger()),
		NSManager:  router.NewNSManager(params),
	}

	return s
}

func (s *serv) join(session router.Session) error {
	defaultRetry := 2
	for i := 0; i < defaultRetry; i++ {
		ns, _ := s.NSManager.GetOrNewNamespace(s.ctx, session.PeerMeta().Domain)
		router, _ := ns.GetOrNewRouter(session.PeerMeta().RouterID)
		if session.PeerMeta().Producer {
			err := router.SetProducer(session)
			if err != nil {
				session.Logger().Debugf("add producer failed: %v", err)
				continue
			}

			break
		} else {
			err := router.AddSubscriber(session)
			if err != nil {
				session.Logger().Debugf("add subscriber failed: %v", err)
				continue
			}

			break
		}
	}
	return nil
}

func (s *serv) Join(ctx context.Context, session router.Session) error {
	h := func(ctx context.Context, req middleware.Request) (interface{}, error) {
		err := s.join(session)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	if next := s.middleware.Match("join"); len(next) > 0 {
		_, err := middleware.Chain(next...)(h)(ctx, middleware.Request{
			Operation: "join",
			Params:    session,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *serv) Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *serv) {
		for _, middleware := range m {
			s.middleware.Use(middleware)
		}
	}
}

func (s *serv) Use(selector string, m ...middleware.Middleware) ServerOption {
	return func(s *serv) {
		for _, middleware := range m {
			s.middleware.Add(selector, middleware)
		}
	}
}
