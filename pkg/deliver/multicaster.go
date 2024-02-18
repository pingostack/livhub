package deliver

import "context"

type MediaFrameMulticaster interface {
	FrameDestination
	FrameSource
	MediaFrameMulticasterClose()
}

type MediaFrameMulticasterImpl struct {
	FrameDestination
	FrameSource
}

func NewMediaFrameMulticaster(ctx context.Context, audioCodec FrameCodec, videoCodec FrameCodec, inPacketType, outPacketType PacketType) MediaFrameMulticaster {
	m := &MediaFrameMulticasterImpl{
		FrameSource:      NewFrameSourceImpl(ctx, audioCodec, videoCodec, inPacketType),
		FrameDestination: NewFrameDestinationImpl(ctx, audioCodec, videoCodec, outPacketType),
	}

	return m
}

func (m *MediaFrameMulticasterImpl) OnFrame(frame Frame, attr Attributes) {
	m.DeliverFrame(frame, attr)
}

func (m *MediaFrameMulticasterImpl) OnFeedback(fb FeedbackMsg) {
	m.DeliverFeedback(fb)
}

func (m *MediaFrameMulticasterImpl) MediaFrameMulticasterClose() {
	m.FrameSource.FrameSourceClose()
	m.FrameDestination.FrameDestinationClose()
}
