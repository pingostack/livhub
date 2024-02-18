package deliver

import (
	"context"
	"fmt"
	"sync"
)

type FrameSourceReceiver interface {
	OnFeedback(fb FeedbackMsg)
}

type FrameSourceDeliver interface {
	DeliverFrame(frame Frame, attr Attributes) error
	DeliverMetaData(metadata Metadata) error
}

type FrameSource interface {
	FrameSourceDeliver
	FrameSourceReceiver
	AddAudioDestination(dest FrameDestination) error
	AddVideoDestination(dest FrameDestination) error
	AddDataDestination(dest FrameDestination) error
	RemoveAudioDestination(dest FrameDestination) error
	RemoveVideoDestination(dest FrameDestination) error
	RemoveDataDestination(dest FrameDestination) error
	SourceAudioCodec() FrameCodec
	SourceVideoCodec() FrameCodec
	SourcePacketType() PacketType
	FrameSourceClose()
}

type FrameSourceImpl struct {
	audioDests []FrameDestination
	videoDests []FrameDestination
	dataDests  []FrameDestination
	lock       sync.RWMutex
	audioCodec FrameCodec
	videoCodec FrameCodec
	packetType PacketType
	ctx        context.Context
	cancel     context.CancelFunc
	closed     bool
}

func NewFrameSourceImpl(ctx context.Context, audioCodec FrameCodec, videoCodec FrameCodec, packetType PacketType) FrameSource {
	fs := &FrameSourceImpl{
		audioCodec: audioCodec,
		videoCodec: videoCodec,
		packetType: packetType,
		audioDests: make([]FrameDestination, 0),
		videoDests: make([]FrameDestination, 0),
		dataDests:  make([]FrameDestination, 0),
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

		for _, d := range fs.audioDests {
			d.unsetAudioSource()
		}

		for _, d := range fs.videoDests {
			d.unsetVideoSource()
		}

		for _, d := range fs.dataDests {
			d.unsetDataSource()
		}

		fs.audioDests = nil
		fs.videoDests = nil
		fs.dataDests = nil
	}()

	return fs
}

func (fs *FrameSourceImpl) AddAudioDestination(dest FrameDestination) error {
	fs.lock.Lock()
	defer func() {
		fs.lock.Unlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	dest.SetAudioSource(fs)
	fs.audioDests = append(fs.audioDests, dest)

	return nil
}

func (fs *FrameSourceImpl) RemoveAudioDestination(dest FrameDestination) error {
	fs.lock.Lock()
	defer func() {
		fs.lock.Unlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	for i, d := range fs.audioDests {
		if d == dest {
			fs.audioDests = append(fs.audioDests[:i], fs.audioDests[i+1:]...)
			dest.unsetAudioSource()
			break
		}
	}

	return nil
}

func (fs *FrameSourceImpl) AddVideoDestination(dest FrameDestination) error {
	fs.lock.Lock()
	defer func() {
		fs.lock.Unlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	dest.SetVideoSource(fs)
	fs.videoDests = append(fs.videoDests, dest)

	return nil
}

func (fs *FrameSourceImpl) RemoveVideoDestination(dest FrameDestination) error {
	fs.lock.Lock()
	defer func() {
		fs.lock.Unlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	for i, d := range fs.videoDests {
		if d == dest {
			fs.videoDests = append(fs.videoDests[:i], fs.videoDests[i+1:]...)
			dest.unsetVideoSource()
			break
		}
	}

	return nil
}

func (fs *FrameSourceImpl) AddDataDestination(dest FrameDestination) error {
	fs.lock.Lock()
	defer func() {
		fs.lock.Unlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	dest.SetDataSource(fs)
	fs.dataDests = append(fs.dataDests, dest)

	return nil
}

func (fs *FrameSourceImpl) RemoveDataDestination(dest FrameDestination) error {
	fs.lock.Lock()
	defer func() {
		fs.lock.Unlock()
	}()

	if fs.closed {
		return ErrFrameSourceClosed
	}

	for i, d := range fs.dataDests {
		if d == dest {
			fs.dataDests = append(fs.dataDests[:i], fs.dataDests[i+1:]...)
			dest.unsetDataSource()
			break
		}
	}

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

	if frame.Codec.IsAudio() {
		for _, d := range fs.audioDests {
			d.OnFrame(frame, attr)
		}
	} else if frame.Codec.IsVideo() {
		for _, d := range fs.videoDests {
			d.OnFrame(frame, attr)
		}
	} else if frame.Codec.IsData() {
		for _, d := range fs.dataDests {
			d.OnFrame(frame, attr)
		}
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

	if fb.Type == FeedbackTypeAudio {
		for _, d := range fs.audioDests {
			d.DeliverFeedback(fb)
		}
	} else if fb.Type == FeedbackTypeVideo {
		for _, d := range fs.videoDests {
			d.DeliverFeedback(fb)
		}
	} else if fb.Type == FeedbackTypeData {
		for _, d := range fs.dataDests {
			d.DeliverFeedback(fb)
		}
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

	if m, ok := metadata.(AudioMetadata); ok {
		for _, d := range fs.audioDests {
			d.OnAudioMetaData(m)
		}
	} else if m, ok := metadata.(VideoMetadata); ok {
		for _, d := range fs.videoDests {
			d.OnVideoMetaData(m)
		}
	} else if m, ok := metadata.(DataMetadata); ok {
		for _, d := range fs.dataDests {
			d.OnDataMetaData(m)
		}
	}

	return nil
}

func (fs *FrameSourceImpl) SourceAudioCodec() FrameCodec {
	return fs.audioCodec
}

func (fs *FrameSourceImpl) SourceVideoCodec() FrameCodec {
	return fs.videoCodec
}

func (fs *FrameSourceImpl) SourcePacketType() PacketType {
	return fs.packetType
}

func (fs *FrameSourceImpl) FrameSourceClose() {
	fs.cancel()
}
