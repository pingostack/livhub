package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pingostack/neon/pkg/demux"
	"github.com/pingostack/neon/pkg/mux"
	"github.com/pingostack/neon/pkg/transcoder"
)

type Stream interface {
	SetAudioSource(src deliver.FrameSource) error
	SetVideoSource(src deliver.FrameSource) error
	SetDataSource(src deliver.FrameSource) error
	UnsetAudioSource()
	UnsetVideoSource()
	UnsetDataSource()
	AddFrameDestination(dest deliver.FrameDestination) error
	Close()
}

type StreamImpl struct {
	ctx              context.Context
	cancel           context.CancelFunc
	audioSrc         deliver.FrameSource
	videoSrc         deliver.FrameSource
	dataSrc          deliver.FrameSource
	demux            map[string]demux.MediaDemux
	audioTranscoders map[string]transcoder.Transcoder
	videoTranscoders map[string]transcoder.Transcoder
	mux              map[string]mux.MediaMux
	lock             sync.RWMutex
	closed           bool
}

func NewStreamImpl(ctx context.Context) Stream {
	s := &StreamImpl{
		demux:            make(map[string]demux.MediaDemux),
		audioTranscoders: make(map[string]transcoder.Transcoder),
		videoTranscoders: make(map[string]transcoder.Transcoder),
		mux:              make(map[string]mux.MediaMux),
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	go func() {
		<-ctx.Done()
		s.lock.Lock()
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("Stream panic %v", err)
			}

			s.lock.Unlock()
		}()

		s.closed = true

		for _, d := range s.demux {
			d.Close()
		}
		for _, t := range s.audioTranscoders {
			t.Close()
		}
		for _, t := range s.videoTranscoders {
			t.Close()
		}
		for _, m := range s.mux {
			m.Close()
		}
	}()

	return s
}

func (s *StreamImpl) SetAudioSource(src deliver.FrameSource) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return ErrStreamClosed
	}

	s.audioSrc = src

	return nil
}

func (s *StreamImpl) SetVideoSource(src deliver.FrameSource) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return ErrStreamClosed
	}

	s.videoSrc = src

	return nil
}

func (s *StreamImpl) SetDataSource(src deliver.FrameSource) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return ErrStreamClosed
	}

	s.dataSrc = src

	return nil
}

func (s *StreamImpl) UnsetAudioSource() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed || s.audioSrc == nil {
		return
	}

	s.audioSrc.FrameSourceClose()
	s.audioSrc = nil

}

func (s *StreamImpl) UnsetVideoSource() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed || s.videoSrc == nil {
		return
	}

	s.videoSrc.FrameSourceClose()
	s.videoSrc = nil
}

func (s *StreamImpl) UnsetDataSource() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed || s.dataSrc == nil {
		return
	}

	s.dataSrc.FrameSourceClose()
	s.dataSrc = nil
}

func (s *StreamImpl) AddFrameDestination(dest deliver.FrameDestination) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	// TODO: add mux transcoder and demux
	if s.audioSrc != nil {
		if s.audioSrc.SourceAudioCodec() != dest.DestinationAudioCodec() {
			return ErrStreamCodecMismatch
		}
		s.audioSrc.AddAudioDestination(dest)
	}

	if s.videoSrc != nil {
		if s.videoSrc.SourceVideoCodec() != dest.DestinationVideoCodec() {
			return ErrStreamCodecMismatch
		}
		s.videoSrc.AddVideoDestination(dest)
	}

	if s.dataSrc != nil {
		s.dataSrc.AddDataDestination(dest)
	}

	return nil
}

func (s *StreamImpl) Close() {
	s.cancel()
}
