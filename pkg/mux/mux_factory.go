package mux

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

func NewMediaMux(ctx context.Context, audioCodec, videoCodec deliver.FrameCodec, inPacketType deliver.PacketType) (MediaMux, error) {
	if inPacketType == deliver.PacketTypeRaw {
		return NewNoopMux(ctx, audioCodec, inPacketType, inPacketType), nil
	} else {
		return nil, ErrMuxNotSupported
	}
}
