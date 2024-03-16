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
	SetFrameSource(src deliver.FrameSource) error
	UnsetSource()
	AddFrameDestination(dest deliver.FrameDestination) error
	Close()
}

type StreamImpl struct {
	ctx              context.Context
	cancel           context.CancelFunc
	src              deliver.FrameSource
	demux            demux.MediaDemux
	audioTranscoders map[string]transcoder.Transcoder
	videoTranscoders map[string]transcoder.Transcoder
	fmts             map[string]StreamFormat
	lock             sync.RWMutex
	closed           bool
	pendingDests     []deliver.FrameDestination
}

func NewStreamImpl(ctx context.Context) Stream {
	s := &StreamImpl{
		audioTranscoders: make(map[string]transcoder.Transcoder),
		videoTranscoders: make(map[string]transcoder.Transcoder),
		fmts:             make(map[string]StreamFormat),
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

		if s.demux != nil {
			s.demux.Close()
		}

		for _, t := range s.audioTranscoders {
			t.Close()
		}

		for _, t := range s.videoTranscoders {
			t.Close()
		}

		for _, f := range s.fmts {
			f.Close()
		}
	}()

	return s
}

func needDemux(src deliver.FrameSource, dest deliver.FrameDestination) bool {
	if src == nil || dest == nil {
		return false
	}

	if src.SourcePacketType() != dest.DestinationPacketType() {
		return true
	}

	if src.SourceAudioCodec() != dest.DestinationAudioCodec() && src.SourceAudioCodec() != deliver.CodecTypeNone && dest.DestinationAudioCodec() != deliver.CodecTypeNone {
		return true
	}

	if src.SourceVideoCodec() != dest.DestinationVideoCodec() && src.SourceVideoCodec() != deliver.CodecTypeNone && dest.DestinationVideoCodec() != deliver.CodecTypeNone {
		return true
	}

	return false
}

func needAudioTranscoder(src deliver.FrameSource, dest deliver.FrameDestination) bool {
	if src == nil || dest == nil {
		return false
	}

	if src.SourceAudioCodec() != dest.DestinationAudioCodec() && src.SourceAudioCodec() != deliver.CodecTypeNone && dest.DestinationAudioCodec() != deliver.CodecTypeNone {
		return true
	}

	return false
}

func needVideoTranscoder(src deliver.FrameSource, dest deliver.FrameDestination) bool {
	if src == nil || dest == nil {
		return false
	}

	if src.SourceVideoCodec() != dest.DestinationVideoCodec() && src.SourceVideoCodec() != deliver.CodecTypeNone && dest.DestinationVideoCodec() != deliver.CodecTypeNone {
		return true
	}

	return false
}

func addFrameDestination(src deliver.FrameSource, dest deliver.FrameDestination) error {
	if src == nil || dest == nil {
		return ErrNilFrameSource
	}

	if err := src.AddAudioDestination(dest); err != nil {
		return err
	}

	if err := src.AddVideoDestination(dest); err != nil {
		return err
	}

	if err := src.AddDataDestination(dest); err != nil {
		return err
	}

	return nil
}

func (s *StreamImpl) linkSrcToDest(src deliver.FrameSource, dest deliver.FrameDestination) error {
	var fmt StreamFormat
	if !needDemux(src, dest) {
		//fmt = NewStreamFormat(s.ctx, dest.DestinationAudioCodec(), dest.DestinationVideoCodec(), dest.DestinationPacketType())

		if err := addFrameDestination(src, dest); err != nil {
			//			fmt.Close()
			return err
		}
		// s.fmts[fmt.PacketType().String()] = fmt

		// if err := fmt.AddDestination(dest); err != nil {
		// 	return err
		// }

		return nil
	}

	if s.demux == nil {
		demux, err := demux.NewMediaDemux(s.ctx, demux.MediaDemuxParams{
			ACodec:       src.SourceAudioCodec(),
			VCodec:       src.SourceVideoCodec(),
			InPacketType: src.SourcePacketType(),
		})
		if err != nil {
			return err
		}

		if err := addFrameDestination(src, demux); err != nil {
			demux.Close()
			return err
		}
		s.demux = demux
	}

	var audioTranscoder, videoTranscoder transcoder.Transcoder
	var err error

	if !needAudioTranscoder(src, s.demux) {
		if err := s.demux.AddAudioDestination(dest); err != nil {
			return err
		}
	} else {
		if _, ok := s.audioTranscoders[dest.DestinationAudioCodec().String()]; !ok {
			audioTranscoder, err = transcoder.NewTranscoder(s.ctx, src.SourceAudioCodec(), dest.DestinationAudioCodec())
			if err != nil {
				return err
			}

			s.audioTranscoders[dest.DestinationAudioCodec().String()] = audioTranscoder

			if err := s.demux.AddAudioDestination(audioTranscoder); err != nil {
				audioTranscoder.Close()
				return err
			}
		} else {
			audioTranscoder = s.audioTranscoders[dest.DestinationAudioCodec().String()]
		}
	}

	if !needVideoTranscoder(src, s.demux) {
		if err := s.demux.AddVideoDestination(dest); err != nil {
			return err
		}
	} else {
		if _, ok := s.videoTranscoders[dest.DestinationVideoCodec().String()]; !ok {
			videoTranscoder, err = transcoder.NewTranscoder(s.ctx, src.SourceVideoCodec(), dest.DestinationVideoCodec())
			if err != nil {
				return err
			}

			s.videoTranscoders[dest.DestinationVideoCodec().String()] = videoTranscoder

			if err := s.demux.AddVideoDestination(videoTranscoder); err != nil {
				videoTranscoder.Close()
				return err
			}
		} else {
			videoTranscoder = s.videoTranscoders[dest.DestinationVideoCodec().String()]
		}
	}

	if _, ok := s.fmts[dest.DestinationPacketType().String()]; ok {
		fmt = s.fmts[dest.DestinationPacketType().String()]
	} else {
		mux, err := mux.NewMediaMux(s.ctx, dest.DestinationAudioCodec(), dest.DestinationVideoCodec(), dest.DestinationPacketType())
		if err != nil {
			return err
		}

		if audioTranscoder != nil && videoTranscoder != nil {
			fmt = NewStreamFormat(s.ctx, dest.DestinationAudioCodec(),
				dest.DestinationVideoCodec(),
				dest.DestinationPacketType(),
				WithAudioTranscoder(audioTranscoder),
				WithVideoTranscoder(videoTranscoder),
				WithMux(mux))
		} else if audioTranscoder != nil {
			fmt = NewStreamFormat(s.ctx, dest.DestinationAudioCodec(),
				dest.DestinationVideoCodec(),
				dest.DestinationPacketType(),
				WithAudioTranscoder(audioTranscoder),
				WithMux(mux))
		} else if videoTranscoder != nil {
			fmt = NewStreamFormat(s.ctx, dest.DestinationAudioCodec(),
				dest.DestinationVideoCodec(),
				dest.DestinationPacketType(),
				WithVideoTranscoder(videoTranscoder),
				WithMux(mux))
		} else {
			fmt = NewStreamFormat(s.ctx, dest.DestinationAudioCodec(),
				dest.DestinationVideoCodec(),
				dest.DestinationPacketType(),
				WithMux(mux))
		}

		s.fmts[dest.DestinationPacketType().String()] = fmt
	}

	if err := fmt.AddDestination(dest); err != nil {
		fmt.Close()
		return err
	}

	return nil
}

func (s *StreamImpl) SetFrameSource(src deliver.FrameSource) error {
	if src == nil {
		return ErrNilFrameSource
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return ErrStreamClosed
	}

	s.src = src

	for _, dest := range s.pendingDests {
		if err := s.linkSrcToDest(src, dest); err != nil {
			continue
		}
	}

	s.pendingDests = nil

	return nil
}

func (s *StreamImpl) UnsetSource() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed || s.src == nil {
		return
	}

	s.src.FrameSourceClose()
	s.src = nil
}

func (s *StreamImpl) AddFrameDestination(dest deliver.FrameDestination) error {
	if dest == nil {
		return ErrNilFrameDestination
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	if s.src != nil {
		return s.linkSrcToDest(s.src, dest)
	} else {
		s.pendingDests = append(s.pendingDests, dest)
	}

	return nil
}

func (s *StreamImpl) RemoveFrameDestination(dest deliver.FrameDestination) error {
	if dest == nil {
		return ErrNilFrameDestination
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	if len(s.fmts) == 0 {
		return nil
	}

	if fmt, ok := s.fmts[dest.DestinationPacketType().String()]; ok {
		if err := fmt.RemoveDestination(dest); err != nil {
			return err
		}

		if fmt.DestinationCount() == 0 {
			fmt.Close()
			delete(s.fmts, dest.DestinationPacketType().String())
		}
	}

	return nil
}

func (s *StreamImpl) Close() {
	s.cancel()
}
