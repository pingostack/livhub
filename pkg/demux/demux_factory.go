package demux

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

type MediaDemuxParams struct {
	ACodec       deliver.CodecType
	VCodec       deliver.CodecType
	InPacketType deliver.PacketType
}

func NewMediaDemux(ctx context.Context, params MediaDemuxParams) (MediaDemux, error) {
	if params.InPacketType == deliver.PacketTypeRaw {
		return NewNoopDemux(ctx, params.ACodec, params.VCodec, params.InPacketType), nil
	} else {
		return nil, ErrDemuxNotSupported
	}
}
