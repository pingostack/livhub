package demux

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

type MediaDemux interface {
	deliver.FrameDestination
	deliver.FrameSource
	Label() string
	Close()
}

type NoopDemux struct {
	deliver.MediaFrameMulticaster
}

func NewNoopDemux(ctx context.Context, audioCodec, videoCodec deliver.FrameCodec, inPacketType deliver.PacketType) *NoopDemux {
	return &NoopDemux{
		MediaFrameMulticaster: deliver.NewMediaFrameMulticaster(ctx, audioCodec, videoCodec, inPacketType, inPacketType),
	}
}

func (m *NoopDemux) Label() string {
	return m.SourceAudioCodec().String() + "-" + m.SourceVideoCodec().String() + "-" + m.SourcePacketType().String() + "-" + m.DestinationPacketType().String()
}

func (m *NoopDemux) Close() {
	m.MediaFrameMulticaster.MediaFrameMulticasterClose()
}
