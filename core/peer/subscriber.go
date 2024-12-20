package peer

import "github.com/pingostack/livhub/pkg/avframe"

type Subscriber interface {
	avframe.WriteCloser
	// ID returns the unique identifier of the subscriber
	ID() string
	Format() avframe.FmtType
	AudioCodecSupported() []avframe.CodecType
	VideoCodecSupported() []avframe.CodecType
	OnActive(processor avframe.Processor)
}
