package pms

import (
	"github.com/pingostack/neon/internal/core/router"
	"github.com/sirupsen/logrus"
)

type Session struct {
	*router.SessionImpl
}

func NewSession(id string, logger *logrus.Entry) *Session {
	return &Session{
		SessionImpl: router.NewSession(id, logger.WithFields(logrus.Fields{
			"session": id,
		})),
	}
}

func (s *Session) Publish(req *router.PublishReq) error {
	return nil
}

func (s *Session) Play(req *router.PlayReq) error {
	return nil
}
