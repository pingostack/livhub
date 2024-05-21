package transcoder

import "errors"

var (
	// ErrTranscoderNotSupported is returned when the transcoder is not supported
	ErrTranscoderNotSupported = errors.New("transcoder not supported")
)
