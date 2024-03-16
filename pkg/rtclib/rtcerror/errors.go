package rtcerror

import "errors"

var (
	ErrAddIceCandidate = errors.New("add ICE candidate error")
	ErrEventNoSCTP     = errors.New("no SCTP")
	ErrNoDTLSTransport = errors.New("no DTLS transport")
	ErrNoICETransport  = errors.New("no ICE transport")
	ErrNoAnswer        = errors.New("no answer")
	ErrICETimeout      = errors.New("ice timeout")
	ErrPanics          = errors.New("panics")
	ErrSdpUnmarshal    = errors.New("sdp unmarshal error")
	ErrInvalidRtpmap   = errors.New("invalid rtpmap")
	ErrNoPayload       = errors.New("no payload type found")
	ErrNoRtxPayload    = errors.New("no rtx payload type found")
	ErrNoCodecForPT    = errors.New("no codec for payload type")
	ErrInvalidFmtp     = errors.New("invalid fmtp")
)
