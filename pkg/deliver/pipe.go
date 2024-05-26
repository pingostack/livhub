package deliver

import (
	"context"

	"github.com/gogf/gf/util/guid"
)

// in-endpoint -> pipe[FrameDestination -> FrameSource] -> out-endpoint

type MediaFramePipeImpl struct {
	FrameDestination
	FrameSource
	ctx    context.Context
	cancel context.CancelFunc
	id     string
}

func NewMediaFramePipe(ctx context.Context, fmtSettings FormatSettings) MediaFramePipe {
	m := &MediaFramePipeImpl{
		id: guid.S(),
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.FrameDestination = NewFrameDestinationImpl(m.ctx, fmtSettings)
	m.FrameSource = NewFrameSourceImpl(m.ctx, Metadata{})

	return m
}

func (m *MediaFramePipeImpl) AddDestination(dest FrameDestination) error {
	return AddDestination(m, dest)
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

func (m *MediaFramePipeImpl) OnMetaData(metadata *Metadata) {
	m.FrameDestination.OnMetaData(metadata)
	m.FrameSource.DeliverMetaData(*metadata)
}

func (m *MediaFramePipeImpl) Close() {
	m.FrameSource.Close()
	m.FrameDestination.Close()
	m.cancel()
}

func (m *MediaFramePipeImpl) Context() context.Context {
	return m.FrameSource.Context()
}

func (m *MediaFramePipeImpl) ID() string {
	return m.id
}

func (m *MediaFramePipeImpl) FormatSettings() FormatSettings {
	return m.FrameDestination.FormatSettings()
}

func (m *MediaFramePipeImpl) Format() string {
	return m.FrameDestination.Format()
}
