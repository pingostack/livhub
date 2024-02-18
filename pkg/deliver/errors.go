package deliver

import "errors"

var (
	ErrFrameSourceClosed      = errors.New("frame source closed")
	ErrFrameDestinationClosed = errors.New("frame destination closed")
)
