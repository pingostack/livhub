package deliver

import "context"

type EnableClose interface {
	Close()
}

type EnableMetaData interface {
	Metadata() *Metadata
}

type FrameDestinationReceiver interface {
	OnFrame(frame Frame, attr Attributes)
	OnMetaData(metadata *Metadata)
}

type FrameDestinationDeliver interface {
	DeliverFeedback(fb FeedbackMsg) error
	OnSource(src FrameSource) error
	unsetSource()
}

type Context interface {
	Context() context.Context
}

type FrameDestination interface {
	Context
	FrameDestinationReceiver
	FrameDestinationDeliver
	EnableClose
	EnableMetaData
}

type FrameSourceReceiver interface {
	OnFeedback(fb FeedbackMsg)
}

type FrameSourceDeliver interface {
	addDestination(dest FrameDestination) error
	DeliverFrame(frame Frame, attr Attributes) error
	DeliverMetaData(metadata Metadata) error
	DestinationCount() int
	AddDestination(dest FrameDestination) error
	RemoveDestination(dest FrameDestination) error
}

type FrameSource interface {
	Context
	FrameSourceDeliver
	FrameSourceReceiver
	EnableClose
	EnableMetaData
}

type MediaFramePipe interface {
	Context
	FrameDestinationReceiver
	FrameDestinationDeliver
	FrameSourceDeliver
	FrameSourceReceiver
	EnableClose
	EnableMetaData
}
