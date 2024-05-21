package demux

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

func NewMediaDemux(ctx context.Context, md deliver.Metadata) (MediaDemux, error) {
	if md.PacketType == deliver.PacketTypeRaw {
		return NewNoopDemux(ctx, md), nil
	} else {
		return nil, ErrDemuxNotSupported
	}
}
