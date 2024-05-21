package deliver

import (
	"context"
	"fmt"
	"sync"
)

type FrameDestinationImpl struct {
	src      FrameSource
	lock     sync.RWMutex
	metadata Metadata
	ctx      context.Context
	cancel   context.CancelFunc
	closed   bool
}

func NewFrameDestinationImpl(ctx context.Context, metadata Metadata) FrameDestination {
	fd := &FrameDestinationImpl{
		metadata: metadata,
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
