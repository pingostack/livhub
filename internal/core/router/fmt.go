package router

import (
	"context"

	sourcemanager "github.com/pingostack/neon/internal/core/router/source_manager"
	"github.com/pingostack/neon/pkg/deliver"
)

type StreamFormat interface {
	deliver.MediaFramePipe
}

type StreamFormatImpl struct {
	deliver.MediaFramePipe
	ctx    context.Context
	cancel context.CancelFunc
	md     deliver.Metadata
	sm     *sourcemanager.Instance
}

type StreamFormatOption func(*StreamFormatImpl)

func WithFrameSourceManager(sm *sourcemanager.Instance) StreamFormatOption {
	return func(fmt *StreamFormatImpl) {
		fmt.sm = sm
	}
}

func NewStreamFormat(ctx context.Context, md deliver.Metadata, opts ...StreamFormatOption) (StreamFormat, error) {
	fmt := &StreamFormatImpl{
		md: md,
	}

	fmt.ctx, fmt.cancel = context.WithCancel(ctx)

	ctx = fmt.ctx

	for _, opt := range opts {
		opt(fmt)
	}

	if fmt.sm.DefaultSource() == nil {
		return nil, ErrNilFrameSource
	}

	fmt.MediaFramePipe = deliver.NewMediaFramePipe(ctx, *fmt.sm.DefaultSource().Metadata(), md)

	deliver.AddDestination(fmt.sm.DefaultSource(), fmt)

	return fmt, nil
}

func (fmt *StreamFormatImpl) AddDestination(dest deliver.FrameDestination) error {
	return deliver.AddDestination(fmt, dest)
}

func (fmt *StreamFormatImpl) RemoveDestination(dest deliver.FrameDestination) error {

	return fmt.MediaFramePipe.RemoveDestination(dest)
}

func (fmt *StreamFormatImpl) OnFeedback(feedback deliver.FeedbackMsg) {
	fmt.MediaFramePipe.OnFeedback(feedback)
}

func (fmt *StreamFormatImpl) Close() {
	fmt.MediaFramePipe.Close()
	fmt.cancel()
}

func (fmt *StreamFormatImpl) Metadata() *deliver.Metadata {
	return &fmt.md
}

func (fmt *StreamFormatImpl) DeliverFeedback(fb deliver.FeedbackMsg) error {
	return fmt.MediaFramePipe.DeliverFeedback(fb)
}
