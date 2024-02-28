package pms

import (
	"context"

	"github.com/pingostack/neon/internal/core"
	"github.com/pingostack/neon/internal/core/router"
	"github.com/sirupsen/logrus"
)

type Session struct {
	router.Session
}

func NewSession(ctx context.Context, pm router.PeerParams, logger *logrus.Entry) *Session {
	s := &Session{
		Session: core.NewSession(ctx, pm, logger),
	}

	go s.waitSessionDone()

	return s
}

func (s *Session) waitSessionDone() {
	<-s.Context().Done()
	s.Logger().Infof("session done")
}
