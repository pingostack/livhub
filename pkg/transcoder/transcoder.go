package transcoder

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

type Transcoder interface {
	deliver.FrameDestination
	deliver.FrameSource
	Label() string
	Close()
}

type NoopTranscoder struct {
	deliver.MediaFrameMulticaster
}

func NewNoopTranscoder(ctx context.Context, inCodec deliver.FrameCodec) Transcoder {
	return &NoopTranscoder{
		MediaFrameMulticaster: deliver.NewMediaFrameMulticaster(ctx, inCodec, inCodec, deliver.PacketTypeRaw, deliver.PacketTypeRaw),
	}
}

func (t *NoopTranscoder) Label() string {
	return t.SourceVideoCodec().String() + "-" + t.SourceAudioCodec().String() + "-" + t.DestinationVideoCodec().String() + "-" + t.DestinationAudioCodec().String()
}

func (t *NoopTranscoder) Close() {
	t.MediaFrameMulticaster.MediaFrameMulticasterClose()
}
