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
	deliver.MediaFramePipe
}

func NewNoopMux(ctx context.Context, md deliver.Metadata) *NoopMux {
	return &NoopMux{
		MediaFramePipe: deliver.NewMediaFramePipe(ctx, md, md),
	}
}

func (m *NoopMux) Close() {
	m.MediaFramePipe.Close()
}
