package mux

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

type MediaMux interface {
	deliver.FrameDestination
	deliver.FrameSource

	Close()
}

type NoopMux struct {
	deliver.MediaFrameMulticaster
	codec         deliver.FrameCodec
	inPacketType  deliver.PacketType
	outPacketType deliver.PacketType
}

func NewNoopMux(ctx context.Context, codec deliver.FrameCodec, inPacketType, outPacketType deliver.PacketType) *NoopMux {
	return &NoopMux{
		MediaFrameMulticaster: deliver.NewMediaFrameMulticaster(ctx, codec, codec, inPacketType, outPacketType),
		codec:                 codec,
		inPacketType:          inPacketType,
		outPacketType:         outPacketType,
	}
}

func (m *NoopMux) Close() {
	m.MediaFrameMulticaster.MediaFrameMulticasterClose()
}
