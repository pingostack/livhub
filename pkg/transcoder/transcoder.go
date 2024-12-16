package transcoder

import (
	"context"

	"github.com/pingostack/livhub/pkg/deliver"
)

type Transcoder interface {
	deliver.FrameDestination
	deliver.FrameSource
	Label() string
	Close()
}

type NoopTranscoder struct {
	deliver.MediaFramePipeImpl
}

func NewNoopTranscoder(ctx context.Context, inCodec deliver.CodecType) Transcoder {
	return &NoopTranscoder{}
}

func (t *NoopTranscoder) Label() string {
	return ""
}

func (t *NoopTranscoder) Close() {
	t.MediaFramePipeImpl.Close()
}
