package core

import "errors"

var (
	ErrFrameSourceAlreadyBound = errors.New("frame source already bound")
	ErrFrameDestinationBound   = errors.New("frame destination already bound")
	ErrFrameSourceNil          = errors.New("frame source is nil")
	ErrFrameDestinationNil     = errors.New("frame destination is nil")
	ErrFrameSourceClosed       = errors.New("frame source closed")
	ErrFrameDestinationClosed  = errors.New("frame destination closed")
)
