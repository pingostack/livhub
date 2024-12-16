package router

import (
	"context"

	sourcemanager "github.com/pingostack/livhub/internal/core/router/source_manager"
	"github.com/pingostack/livhub/pkg/deliver"
)

type StreamFormat interface {
	deliver.MediaFramePipe
}

type StreamFormatImpl struct {
	deliver.MediaFramePipe
	ctx    context.Context
	cancel context.CancelFunc
	sm     *sourcemanager.Instance
}

type StreamFormatOption func(*StreamFormatImpl)

func WithFrameSourceManager(sm *sourcemanager.Instance) StreamFormatOption {
	return func(fmt *StreamFormatImpl) {
		fmt.sm = sm
	}
}

func NewStreamFormat(ctx context.Context, fmtSettings deliver.FormatSettings, opts ...StreamFormatOption) (StreamFormat, error) {
	fmt := &StreamFormatImpl{}

	fmt.ctx, fmt.cancel = context.WithCancel(ctx)

	ctx = fmt.ctx

	for _, opt := range opts {
		opt(fmt)
	}

	if fmt.sm.DefaultSource() == nil {
		return nil, ErrNilFrameSource
	}

	fmt.MediaFramePipe = deliver.NewMediaFramePipe(ctx, fmtSettings)

	deliver.AddDestination(fmt.sm.DefaultSource(), fmt)

	return fmt, nil
}

func (fmt *StreamFormatImpl) Close() {
	fmt.MediaFramePipe.Close()
	fmt.cancel()
}
