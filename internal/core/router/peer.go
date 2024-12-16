package router

import (
	"context"

	"github.com/pingostack/livhub/pkg/deliver"
	"github.com/sirupsen/logrus"
)

type PeerParams struct {
	ACodec         deliver.CodecType  `json:"audio_codec"`
	VCodec         deliver.CodecType  `json:"video_codec"`
	PacketType     deliver.PacketType `json:"packet_type"`
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
	ID() string
	Set(key, value interface{})
	Get(key interface{}) interface{}
	Logger() *logrus.Entry
	Context() context.Context
	GetNamespace() *Namespace
	SetNamespace(ns *Namespace)
	GetRouter() Router
	SetRouter(r Router)
	PeerParams() PeerParams
	Finalize(e error)
	RouterID() string
	BindFrameSource(src deliver.FrameSource) error
	BindFrameDestination(dest deliver.FrameDestination) error
	FrameSource() deliver.FrameSource
	FrameDestination() deliver.FrameDestination
	Join() error
}
