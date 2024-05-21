package mux

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
)

func NewMediaMux(ctx context.Context, md deliver.Metadata) (MediaMux, error) {
	if md.PacketType == deliver.PacketTypeRaw {
		return NewNoopMux(ctx, md), nil
	} else {
		return nil, ErrMuxNotSupported
	}
}
