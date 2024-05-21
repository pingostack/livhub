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
	deliver.MediaFramePipe
}

func NewNoopDemux(ctx context.Context, md deliver.Metadata) *NoopDemux {
	return &NoopDemux{
		MediaFramePipe: deliver.NewMediaFramePipe(ctx, md, md),
	}
}

func (m *NoopDemux) Label() string {
	return ""
}

func (m *NoopDemux) Close() {
	m.MediaFramePipe.Close()
}
