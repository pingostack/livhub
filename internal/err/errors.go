package err

import "errors"

var (
	ErrNSNotFound           = errors.New("namespace not found")
	ErrRouterNotFound       = errors.New("router not found")
	ErrSessionAlreadyExists = errors.New("session already exists")
	ErrRouterClosed         = errors.New("router closed")
	ErrSessionIdleTimeout   = errors.New("session idle timeout")
	ErrProducerEmpty        = errors.New("producer empty")
	ErrProducerRepeated     = errors.New("producer repeated")
)
