package core

import (
	"context"
	"errors"

	"github.com/pingostack/neon/internal/core/middleware"
	"github.com/pingostack/neon/internal/core/router"
	"github.com/pingostack/neon/pkg/eventemitter"
)

type serv struct {
	*router.NSManager
	middleware middleware.Matcher
	ee         *eventemitter.EventEmitter
}

type ServerOption func(*serv)

var (
	defaultServ = NewServ(context.Background())
)

const (
	defaultEventEmitterSize = 100
)

func NewServ(ctx context.Context) *serv {
	s := &serv{
		middleware: middleware.New(),
		ee:         eventemitter.NewEventEmitter(ctx, defaultEventEmitterSize, DefaultLogger()),
		NSManager:  router.NewNSManager(),
	}

	return s
}

func (s *serv) join(info *router.PeerInfo) error {
	ns := s.NSManager.GetOrNewNamespace(info.Domain)
	router, _ := ns.GetOrNewRouter(info.URI)
	ok := router.AddSessionIfNotExists(info.Session)
	if !ok {
		return errors.New("session already exists")
	}

	info.Session.SetRouter(router)

	return nil
}

func (s *serv) Publish(req *router.PublishReq) error {
	h := func(ctx context.Context, r interface{}) (interface{}, error) {
		err := s.join(&req.PeerInfo)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	if next := s.middleware.Match("publish"); len(next) > 0 {
		_, err := middleware.Chain(next...)(h)(req.Ctx, req)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *serv) Play(req *router.PlayReq) error {
	err := s.join(&req.PeerInfo)
	if err != nil {
		return err
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

func Publish(req *router.PublishReq) error {
	return defaultServ.Publish(req)
}

func Play(req *router.PlayReq) error {
	return defaultServ.Play(req)
}
