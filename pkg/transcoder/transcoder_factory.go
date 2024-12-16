package transcoder

import (
	"context"

	"github.com/pingostack/livhub/pkg/deliver"
)

func NewTranscoder(ctx context.Context, inCodec, outCodec deliver.CodecType) (Transcoder, error) {
	if inCodec == outCodec {
		return NewNoopTranscoder(ctx, inCodec), nil
	} else {
		return nil, ErrTranscoderNotSupported
	}
}
