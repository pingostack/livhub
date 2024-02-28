package router

import (
	"context"

	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pingostack/neon/pkg/mux"
	"github.com/pingostack/neon/pkg/transcoder"
)

type StreamFormat interface {
	AddDestination(dest deliver.FrameDestination) error
	RemoveDestination(dest deliver.FrameDestination) error
	Close()
	PacketType() deliver.PacketType
	DestinationCount() int
}

type StreamFormatImpl struct {
	ctx             context.Context
	audioTranscoder transcoder.Transcoder
	videoTranscoder transcoder.Transcoder
	mux             mux.MediaMux
	acodec          deliver.CodecType
	vcodec          deliver.CodecType
	outPacketType   deliver.PacketType
}

func WithAudioTranscoder(transcoder transcoder.Transcoder) func(fmt *StreamFormatImpl) {
	return func(fmt *StreamFormatImpl) {
		fmt.audioTranscoder = transcoder
	}
}

func WithVideoTranscoder(transcoder transcoder.Transcoder) func(fmt *StreamFormatImpl) {
	return func(fmt *StreamFormatImpl) {
		fmt.videoTranscoder = transcoder
	}
}

func WithMux(mux mux.MediaMux) func(fmt *StreamFormatImpl) {
	return func(fmt *StreamFormatImpl) {
		fmt.mux = mux
	}
}

func NewStreamFormat(ctx context.Context, acodec, vcodec deliver.CodecType, outPacketType deliver.PacketType, opts ...func(fmt *StreamFormatImpl)) *StreamFormatImpl {
	fmt := &StreamFormatImpl{
		ctx:           ctx,
		acodec:        acodec,
		vcodec:        vcodec,
		outPacketType: outPacketType,
	}

	for _, opt := range opts {
		opt(fmt)
	}

	if fmt.mux == nil {
		fmt.mux = mux.NewNoopMux(ctx, acodec, vcodec, outPacketType)
	}

	return fmt
}

func (fmt *StreamFormatImpl) AddDestination(dest deliver.FrameDestination) error {
	if err := fmt.mux.AddAudioDestination(dest); err != nil {
		return err
	}

	if err := fmt.mux.AddVideoDestination(dest); err != nil {
		return err
	}

	if err := fmt.mux.AddDataDestination(dest); err != nil {
		return err
	}

	return nil
}

func (fmt *StreamFormatImpl) RemoveDestination(dest deliver.FrameDestination) error {
	if err := fmt.mux.RemoveAudioDestination(dest); err != nil {
		return err
	}

	if err := fmt.mux.RemoveVideoDestination(dest); err != nil {
		return err
	}

	if err := fmt.mux.RemoveDataDestination(dest); err != nil {
		return err
	}

	return nil
}

func (fmt *StreamFormatImpl) Close() {
	if fmt.mux != nil {
		fmt.mux.Close()
	}
}

func (fmt *StreamFormatImpl) PacketType() deliver.PacketType {
	return fmt.outPacketType
}

func (fmt *StreamFormatImpl) DestinationCount() int {
	return fmt.mux.DestinationCount()
}
