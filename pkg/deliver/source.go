package deliver

import (
	"context"
	"fmt"
	"sync"
)

type DestinationInfo struct {
	HasAudio bool
	HasVideo bool
	HasData  bool
}

type FrameSourceImpl struct {
	dests     []FrameDestination
	destIndex map[FrameDestination]FrameDestination
	lock      sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	closed    bool
	metadata  Metadata
}

func NewFrameSourceImpl(ctx context.Context, metadata Metadata) FrameSource {
	fs := &FrameSourceImpl{
		metadata:  metadata,
		dests:     make([]FrameDestination, 0),
		destIndex: make(map[FrameDestination]FrameDestination),
	}

	fs.ctx, fs.cancel = context.WithCancel(ctx)

	go func() {
		<-fs.ctx.Done()

		fs.lock.RLock()
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("FrameSource panic %v", err)
			}

			fs.lock.RUnlock()
		}()

		fs.closed = true

		for _, d := range fs.dests {
			d.unsetSource()
		}

		fs.dests = nil
	}()

	return fs
}

func (fs *FrameSourceImpl) addDestination(dest FrameDestination) error {
	fs.lock.Lock()
	defer func() {
		fs.lock.Unlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	if _, ok := fs.destIndex[dest]; ok {
		return ErrFrameDestinationExists
	}

	fs.dests = append(fs.dests, dest)
	fs.destIndex[dest] = dest

	dest.OnMetaData(&fs.metadata)

	return nil
}

func (fs *FrameSourceImpl) AddDestination(dest FrameDestination) error {
	return AddDestination(fs, dest)
}

func (fs *FrameSourceImpl) RemoveDestination(dest FrameDestination) error {
	fs.lock.Lock()
	defer func() {
		fs.lock.Unlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	if !fs.metadata.HasAudio() {
		return ErrFrameSourceAudioNotSupport
	}

	for i, d := range fs.dests {
		if d == dest {
			fs.dests = append(fs.dests[:i], fs.dests[i+1:]...)
			dest.unsetSource()
			break
		}
	}

	delete(fs.destIndex, dest)

	return nil
}

func (fs *FrameSourceImpl) DeliverFrame(frame Frame, attr Attributes) error {
	fs.lock.RLock()
	defer func() {
		fs.lock.RUnlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	for _, d := range fs.dests {
		d.OnFrame(frame, attr)
	}

	return nil
}

func (fs *FrameSourceImpl) OnFeedback(fb FeedbackMsg) {
	fs.lock.RLock()
	defer func() {
		fs.lock.RUnlock()
	}()

	if fs.closed {
		return
	}
}

func (fs *FrameSourceImpl) DeliverMetaData(metadata Metadata) error {
	fs.lock.RLock()
	defer func() {
		fs.lock.RUnlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	fs.metadata = metadata

	for _, d := range fs.dests {
		d.OnMetaData(&metadata)
	}

	return nil
}

func (fs *FrameSourceImpl) Metadata() *Metadata {
	return &fs.metadata
}

func (fs *FrameSourceImpl) Close() {
	fs.cancel()
}

func (fs *FrameSourceImpl) DestinationCount() int {
	fs.lock.RLock()
	defer func() {
		fs.lock.RUnlock()
	}()

	return len(fs.dests)
}

func (fs *FrameSourceImpl) Context() context.Context {
	return fs.ctx
}

func AddDestination[SRC FrameSource, DEST FrameDestination](src SRC, dest DEST) error {
	err := src.addDestination(dest)
	if err != nil {
		return err
	}

	return dest.OnSource(src)
}
