package deliver

import (
	"context"
	"fmt"
	"sync"
)

type FrameDestinationReceiver interface {
	OnFrame(frame Frame, attr Attributes)
	OnAudioMetaData(metadata *AudioMetadata)
	OnVideoMetaData(metadata *VideoMetadata)
	OnDataMetaData(metadata *DataMetadata)
}

type FrameDestinationDeliver interface {
	DeliverFeedback(fb FeedbackMsg) error
}

type FrameDestination interface {
	FrameDestinationReceiver
	FrameDestinationDeliver
	SetAudioSource(src FrameSource) error
	SetVideoSource(src FrameSource) error
	SetDataSource(src FrameSource) error
	unsetAudioSource()
	unsetVideoSource()
	unsetDataSource()
	DestinationAudioCodec() CodecType
	DestinationVideoCodec() CodecType
	DestinationPacketType() PacketType
	FrameDestinationClose()
}

type FrameDestinationImpl struct {
	audioSrc   FrameSource
	videoSrc   FrameSource
	dataSrc    FrameSource
	lock       sync.RWMutex
	acodec     CodecType
	vcodec     CodecType
	packetType PacketType
	ctx        context.Context
	cancel     context.CancelFunc
	closed     bool
}

func NewFrameDestinationImpl(ctx context.Context, acodec, vcodec CodecType, packetType PacketType) FrameDestination {
	fd := &FrameDestinationImpl{
		acodec:     acodec,
		vcodec:     vcodec,
		packetType: packetType,
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
		audioSrc := fd.audioSrc
		videoSrc := fd.videoSrc
		dataSrc := fd.dataSrc
		fd.lock.Unlock()

		if audioSrc != nil {
			audioSrc.RemoveAudioDestination(fd)
		}

		if videoSrc != nil {
			videoSrc.RemoveVideoDestination(fd)
		}

		if dataSrc != nil {
			dataSrc.RemoveDataDestination(fd)
		}
	}()

	return fd
}

func (fd *FrameDestinationImpl) SetAudioSource(src FrameSource) error {
	fd.lock.Lock()
	defer fd.lock.Unlock()

	if fd.closed {
		return ErrFrameDestinationClosed
	}

	fd.audioSrc = src

	return nil
}

func (fd *FrameDestinationImpl) SetVideoSource(src FrameSource) error {
	fd.lock.Lock()
	defer fd.lock.Unlock()

	if fd.closed {
		return ErrFrameDestinationClosed
	}

	fd.videoSrc = src

	return nil
}

func (fd *FrameDestinationImpl) SetDataSource(src FrameSource) error {
	fd.lock.Lock()
	defer fd.lock.Unlock()

	if fd.closed {
		return ErrFrameDestinationClosed
	}

	fd.dataSrc = src

	return nil
}

func (fd *FrameDestinationImpl) unsetAudioSource() {
	fd.lock.Lock()
	defer fd.lock.Unlock()

	if fd.closed {
		return
	}

	fd.audioSrc = nil
}

func (fd *FrameDestinationImpl) unsetVideoSource() {
	fd.lock.Lock()
	defer fd.lock.Unlock()

	if fd.closed {
		return
	}

	fd.videoSrc = nil
}

func (fd *FrameDestinationImpl) unsetDataSource() {
	fd.lock.Lock()
	defer fd.lock.Unlock()

	if fd.closed {
		return
	}

	fd.dataSrc = nil
}

func (fd *FrameDestinationImpl) OnFrame(frame Frame, attr Attributes) {
}

func (fd *FrameDestinationImpl) DeliverFeedback(fb FeedbackMsg) error {
	fd.lock.RLock()
	defer fd.lock.RUnlock()
	if fd.closed {
		return ErrFrameDestinationClosed
	}

	if fb.Type == FeedbackTypeAudio {
		if fd.audioSrc != nil {
			fd.audioSrc.OnFeedback(fb)
		}
	} else if fb.Type == FeedbackTypeVideo {
		if fd.videoSrc != nil {
			fd.videoSrc.OnFeedback(fb)
		}
	} else if fb.Type == FeedbackTypeData {
		if fd.dataSrc != nil {
			fd.dataSrc.OnFeedback(fb)
		}
	}

	return nil
}

func (fd *FrameDestinationImpl) OnData(data []byte, attr Attributes) {
}

func (fd *FrameDestinationImpl) OnAudioMetaData(metadata *AudioMetadata) {
}

func (fd *FrameDestinationImpl) OnVideoMetaData(metadata *VideoMetadata) {
}

func (fd *FrameDestinationImpl) OnDataMetaData(metadata *DataMetadata) {
}

func (fd *FrameDestinationImpl) DestinationAudioCodec() CodecType {
	return fd.acodec
}

func (fd *FrameDestinationImpl) DestinationVideoCodec() CodecType {
	return fd.vcodec
}

func (fd *FrameDestinationImpl) DestinationPacketType() PacketType {
	return fd.packetType
}

func (fd *FrameDestinationImpl) FrameDestinationClose() {
	fd.cancel()
}
