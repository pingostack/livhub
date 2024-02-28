package transport

import "errors"

var (
	ErrAddIceCandidate = errors.New("add ICE candidate error")
	ErrEventNoSCTP     = errors.New("no SCTP")
	ErrNoDTLSTransport = errors.New("no DTLS transport")
	ErrNoICETransport  = errors.New("no ICE transport")
	ErrNoAnswer        = errors.New("no answer")
	ErrICETimeout      = errors.New("ice timeout")
	ErrPanics          = errors.New("panics")
)
