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
	DestinationCount() int
	AddAudioDestination(dest FrameDestination) error
	AddVideoDestination(dest FrameDestination) error
	AddDataDestination(dest FrameDestination) error
	RemoveAudioDestination(dest FrameDestination) error
	RemoveVideoDestination(dest FrameDestination) error
	RemoveDataDestination(dest FrameDestination) error
	SourceAudioCodec() CodecType
	SourceVideoCodec() CodecType
	SourcePacketType() PacketType
	FrameSourceClose()
}

type DestinationInfo struct {
	HasAudio bool
	HasVideo bool
	HasData  bool
}

type FrameSourceImpl struct {
	audioDests     []FrameDestination
	videoDests     []FrameDestination
	dataDests      []FrameDestination
	audioDestIndex map[FrameDestination]FrameDestination
	videoDestIndex map[FrameDestination]FrameDestination
	dataDestIndex  map[FrameDestination]FrameDestination
	destStatistics map[FrameDestination]*DestinationInfo
	lock           sync.RWMutex
	acodec         CodecType
	vcodec         CodecType
	packetType     PacketType
	ctx            context.Context
	cancel         context.CancelFunc
	closed         bool
}

func NewFrameSourceImpl(ctx context.Context, acodec CodecType, vcodec CodecType, packetType PacketType) FrameSource {
	fs := &FrameSourceImpl{
		acodec:         acodec,
		vcodec:         vcodec,
		packetType:     packetType,
		audioDests:     make([]FrameDestination, 0),
		videoDests:     make([]FrameDestination, 0),
		dataDests:      make([]FrameDestination, 0),
		audioDestIndex: make(map[FrameDestination]FrameDestination),
		videoDestIndex: make(map[FrameDestination]FrameDestination),
		dataDestIndex:  make(map[FrameDestination]FrameDestination),
		destStatistics: make(map[FrameDestination]*DestinationInfo),
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
		fs.audioDestIndex = nil
		fs.videoDestIndex = nil
		fs.dataDestIndex = nil
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

	if fs.acodec != dest.DestinationAudioCodec() {
		return ErrCodecNotSupported
	}

	if _, ok := fs.audioDestIndex[dest]; ok {
		return ErrFrameDestinationExists
	}

	if info, ok := fs.destStatistics[dest]; ok {
		info.HasAudio = true
	} else {
		fs.destStatistics[dest] = &DestinationInfo{
			HasAudio: true,
		}
	}

	dest.SetAudioSource(fs)
	fs.audioDests = append(fs.audioDests, dest)
	fs.audioDestIndex[dest] = dest

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

	delete(fs.destStatistics, dest)
	delete(fs.audioDestIndex, dest)

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

	if fs.vcodec != dest.DestinationVideoCodec() {
		return ErrCodecNotSupported
	}

	if _, ok := fs.videoDestIndex[dest]; ok {
		return ErrFrameDestinationExists
	}

	dest.SetVideoSource(fs)
	fs.videoDests = append(fs.videoDests, dest)
	fs.videoDestIndex[dest] = dest
	if info, ok := fs.destStatistics[dest]; ok {
		info.HasVideo = true
	} else {
		fs.destStatistics[dest] = &DestinationInfo{
			HasVideo: true,
		}
	}

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

	delete(fs.videoDestIndex, dest)
	delete(fs.destStatistics, dest)

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

	if _, ok := fs.dataDestIndex[dest]; ok {
		return ErrFrameDestinationExists
	}

	dest.SetDataSource(fs)
	fs.dataDests = append(fs.dataDests, dest)
	fs.dataDestIndex[dest] = dest

	if info, ok := fs.destStatistics[dest]; ok {
		info.HasData = true
	} else {
		fs.destStatistics[dest] = &DestinationInfo{
			HasData: true,
		}
	}

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

	delete(fs.dataDestIndex, dest)
	delete(fs.destStatistics, dest)

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

	if metadata.Audio != nil {
		for _, d := range fs.audioDests {
			d.OnAudioMetaData(metadata.Audio)
		}
	}

	if metadata.Video != nil {
		for _, d := range fs.videoDests {
			d.OnVideoMetaData(metadata.Video)
		}
	}

	if metadata.Data != nil {
		for _, d := range fs.dataDests {
			d.OnDataMetaData(metadata.Data)
		}
	}

	return nil
}

func (fs *FrameSourceImpl) SourceAudioCodec() CodecType {
	return fs.acodec
}

func (fs *FrameSourceImpl) SourceVideoCodec() CodecType {
	return fs.vcodec
}

func (fs *FrameSourceImpl) SourcePacketType() PacketType {
	return fs.packetType
}

func (fs *FrameSourceImpl) FrameSourceClose() {
	fs.cancel()
}

func (fs *FrameSourceImpl) DestinationCount() int {
	fs.lock.RLock()
	defer func() {
		fs.lock.RUnlock()
	}()

	return len(fs.destStatistics)
}
