package demux

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

func NewMediaDemux(ctx context.Context, audioCodec, videoCodec deliver.FrameCodec, inPacketType deliver.PacketType) (MediaDemux, error) {
	if inPacketType == deliver.PacketTypeRaw {
		return NewNoopDemux(ctx, audioCodec, videoCodec, inPacketType), nil
	} else {
		return nil, ErrDemuxNotSupported
	}
}
