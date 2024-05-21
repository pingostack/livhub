package router

import "errors"

var (
	ErrNSNotFound           = errors.New("namespace not found")
	ErrRouterNotFound       = errors.New("router not found")
	ErrSessionAlreadyExists = errors.New("session already exists")
	ErrRouterClosed         = errors.New("router closed")
	ErrSessionIdleTimeout   = errors.New("session idle timeout")
	ErrProducerEmpty        = errors.New("producer empty")
	ErrProducerRepeated     = errors.New("producer repeated")
	ErrStreamFormatNotFound = errors.New("stream format not found")
	ErrStreamClosed         = errors.New("stream closed")
	ErrNilFrameDestination  = errors.New("nil frame destination")
	ErrNilFrameSource       = errors.New("nil frame source")
	ErrStreamTimeout        = errors.New("stream timeout")
	ErrFrameSourceExists    = errors.New("frame source exists")
)
