package mux

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

func NewMediaMux(ctx context.Context,
	acodec, vcodec deliver.CodecType,
	outPacketType deliver.PacketType) (MediaMux, error) {
	if outPacketType == deliver.PacketTypeRaw {
		return NewNoopMux(ctx, acodec, vcodec, outPacketType), nil
	} else {
		return nil, ErrMuxNotSupported
	}
}
