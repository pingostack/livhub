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
}

func NewNoopMux(ctx context.Context, acodec, vcodec deliver.CodecType, packetType deliver.PacketType) *NoopMux {
	return &NoopMux{
		MediaFrameMulticaster: deliver.NewMediaFrameMulticaster(ctx, acodec, vcodec, packetType, packetType),
	}
}

func (m *NoopMux) Close() {
	m.MediaFrameMulticaster.MediaFrameMulticasterClose()
}
