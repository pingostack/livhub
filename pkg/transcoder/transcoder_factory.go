package transcoder

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

func NewTranscoder(ctx context.Context, inCodec, outCodec deliver.FrameCodec) (Transcoder, error) {
	if inCodec == outCodec {
		return NewNoopTranscoder(ctx, inCodec), nil
	} else {
		return nil, ErrTranscoderNotSupported
	}
}
