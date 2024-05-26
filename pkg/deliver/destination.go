package deliver

import (
	"context"
	"fmt"
	"sync"

	"github.com/gogf/gf/util/guid"
)

type FrameDestinationImpl struct {
	src      FrameSource
	lock     sync.RWMutex
	metadata Metadata
	ctx      context.Context
	cancel   context.CancelFunc
	closed   bool
	id       string
	settings FormatSettings
}

func NewFrameDestinationImpl(ctx context.Context, settings FormatSettings) FrameDestination {
	fd := &FrameDestinationImpl{
		settings: settings,
		id:       guid.S(),
	}

	fd.ctx, fd.cancel = context.WithCancel(ctx)

	go func() {
		<-fd.ctx.Done()
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("FrameDestination panic %v", err)
			}
		}()

		fd.lock.Lock()
		fd.closed = true
		src := fd.src
		fd.lock.Unlock()

		if src != nil {
			src.RemoveDestination(fd)
		}
	}()

	return fd
}

func (fd *FrameDestinationImpl) OnSource(src FrameSource) error {
	fd.lock.Lock()
	defer fd.lock.Unlock()

	if fd.closed {
		return ErrFrameDestinationClosed
	}

	fd.src = src

	return nil
}

func (fd *FrameDestinationImpl) unsetSource() {
	fd.lock.Lock()
	defer fd.lock.Unlock()

	if fd.closed {
		return
	}

	fd.src = nil
}

func (fd *FrameDestinationImpl) OnFrame(frame Frame, attr Attributes) {
}

func (fd *FrameDestinationImpl) DeliverFeedback(fb FeedbackMsg) error {
	fd.lock.RLock()
	defer fd.lock.RUnlock()
	if fd.closed {
		return ErrFrameDestinationClosed
	}

	if fd.src != nil {
		fd.src.OnFeedback(fb)
	}

	return nil
}

func (fd *FrameDestinationImpl) OnMetaData(metadata *Metadata) {
	fd.metadata = *metadata
}

func (fd *FrameDestinationImpl) Metadata() *Metadata {
	return &fd.metadata
}

func (fd *FrameDestinationImpl) Close() {
	fd.cancel()
}

func (fd *FrameDestinationImpl) Context() context.Context {
	return fd.ctx
}

func (fd *FrameDestinationImpl) ID() string {
	return fd.id
}

func (fd *FrameDestinationImpl) Format() string {
	return fd.settings.PacketType.String()
}

func (fd *FrameDestinationImpl) FormatSettings() FormatSettings {
	return fd.settings
}
