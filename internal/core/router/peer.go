package router

import (
	"context"

	"github.com/pingostack/neon/pkg/eventemitter"
	"github.com/pingostack/neon/pkg/streaminterceptor"
	"github.com/sirupsen/logrus"
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
	streaminterceptor.Reader
	ID() string
	RouterID() string
	Set(key, value interface{})
	Get(key interface{}) interface{}
	Logger() *logrus.Entry
	Context() context.Context
	GetNamespace() *Namespace
	SetNamespace(ns *Namespace)
	GetRouter() Router
	SetRouter(r Router)
	PeerMeta() PeerMeta
	Finalize(e error)
	Join() error
}
