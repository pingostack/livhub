package deliver

import (
	"context"
)

// in-endpoint -> pipe[FrameDestination -> FrameSource] -> out-endpoint

type MediaFramePipeImpl struct {
	FrameDestination
	FrameSource
	ctx    context.Context
	cancel context.CancelFunc
}

func NewMediaFramePipe(ctx context.Context, inMd Metadata, outMd Metadata) MediaFramePipe {
	m := &MediaFramePipeImpl{}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.FrameDestination = NewFrameDestinationImpl(m.ctx, inMd)
	m.FrameSource = NewFrameSourceImpl(m.ctx, outMd)

	return m
}

func (m *MediaFramePipeImpl) AddDestination(dest FrameDestination) error {
	return AddDestination(m.FrameSource, dest)
}

func (m *MediaFramePipeImpl) OnFrame(frame Frame, attr Attributes) {
	m.FrameSource.DeliverFrame(frame, attr)
}

func (m *MediaFramePipeImpl) OnFeedback(fb FeedbackMsg) {
	m.FrameDestination.DeliverFeedback(fb)
}

func (m *MediaFramePipeImpl) Metadata() *Metadata {
	return m.FrameSource.Metadata()
}

func (m *MediaFramePipeImpl) Close() {
	m.FrameSource.Close()
	m.FrameDestination.Close()
	m.cancel()
}

func (m *MediaFramePipeImpl) Context() context.Context {
	return m.FrameSource.Context()
}
