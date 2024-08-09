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
	ns, _ := s.NSManager.GetOrNewNamespaceByDomain(s.ctx, session.PeerParams().Domain)
	// if ns == nil {
	// 	return router.ErrNamespaceNotFound
	// }

	r, _ := ns.GetOrNewRouter(session.PeerParams().RouterID)
	err := r.AddSession(session)
	if err == router.ErrRouterClosed {
		ns.RemoveRouter(r)
		r, _ = ns.GetOrNewRouter(session.PeerParams().RouterID)
		err = r.AddSession(session)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	session.SetRouter(r)
	session.SetNamespace(ns)

	return nil
}

func (s *serv) Join(ctx context.Context, session router.Session) error {
	//	h := func(ctx context.Context, req middleware.Request) (interface{}, error) {
	err := s.join(session)
	if err != nil {
		return err
	}

	// 	return nil, nil
	// }

	// if next := s.middleware.Match("join"); len(next) > 0 {
	// 	_, err := middleware.Chain(next...)(h)(ctx, middleware.Request{
	// 		Operation: middleware.OperationJoin,
	// 		Params:    session,
	// 	})
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

// func (s *serv) Middleware(m ...middleware.Middleware) ServerOption {
// 	return func(s *serv) {
// 		for _, middleware := range m {
// 			s.middleware.Use(middleware)
// 		}
// 	}
// }

// func (s *serv) Use(selector string, m ...middleware.Middleware) ServerOption {
// 	return func(s *serv) {
// 		for _, middleware := range m {
// 			s.middleware.Add(selector, middleware)
// 		}
// 	}
// }
