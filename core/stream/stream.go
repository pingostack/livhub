package stream

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/pingostack/livhub/core/peer"
	"github.com/pingostack/livhub/core/plugin"
	"github.com/pingostack/livhub/pkg/avframe"
	"github.com/pingostack/livhub/pkg/errcode"
	"github.com/pingostack/livhub/pkg/logger"
	"github.com/pkg/errors"
)

const (
// EventStreamActive eventemitter.EventID = iota
// EventStreamInactive
//
//	EventSubStreamEmpty eventemitter.EventID = iota
)

type SubscribeResultFunc func(sub peer.Subscriber, processor avframe.Processor, err error)
type waitingSub struct {
	sub      peer.Subscriber
	onResult SubscribeResultFunc
}

type StreamSettings struct {
	IdleTimeout time.Duration
}

type Stream struct {
	id     string
	ctx    context.Context
	cancel context.CancelFunc

	noopProcessor *avframe.Pipeline
	demuxer       *avframe.Pipeline
	audioDecoders map[avframe.CodecType]*avframe.Pipeline
	videoDecoders map[avframe.CodecType]*avframe.Pipeline
	audioEncoders map[avframe.CodecType]*avframe.Pipeline
	videoEncoders map[avframe.CodecType]*avframe.Pipeline
	muxers        map[avframe.FmtType]*avframe.Pipeline
	lock          sync.RWMutex

	publisher peer.Publisher

	subStreams map[avframe.FmtType]*subStream

	waitingSubs []waitingSub
	closed      uint32

	//	eventHandle eventemitter.EventEmitter

	logger logger.Logger

	settings           StreamSettings
	chPublisherSwapped chan struct{}
	chWaitPublisher    chan struct{}
}

type StreamOption func(*StreamSettings)

func WithStreamSettings(settings StreamSettings) StreamOption {
	return func(s *StreamSettings) {
		*s = settings
	}
}

const (
	DefaultIdleTimeoutSeconds = 10
)

func NewStream(ctx context.Context, id string, opts ...StreamOption) *Stream {
	ctx, cancel := context.WithCancel(ctx)
	settings := StreamSettings{
		IdleTimeout: time.Second * DefaultIdleTimeoutSeconds,
	}

	for _, opt := range opts {
		opt(&settings)
	}

	stream := &Stream{
		id:            id,
		ctx:           ctx,
		cancel:        cancel,
		audioDecoders: make(map[avframe.CodecType]*avframe.Pipeline),
		videoDecoders: make(map[avframe.CodecType]*avframe.Pipeline),
		audioEncoders: make(map[avframe.CodecType]*avframe.Pipeline),
		videoEncoders: make(map[avframe.CodecType]*avframe.Pipeline),
		muxers:        make(map[avframe.FmtType]*avframe.Pipeline),
		subStreams:    make(map[avframe.FmtType]*subStream),
		waitingSubs:   make([]waitingSub, 0),
		//		eventHandle:     eventemitter.NewEventEmitter(ctx, 1024),
		logger:             logger.WithFields(map[string]interface{}{"stream": id}),
		chPublisherSwapped: make(chan struct{}, 1),
		chWaitPublisher:    make(chan struct{}, 1),
		settings:           settings,
	}

	go stream.run()

	return stream
}

func (s *Stream) GetID() string {
	return s.id
}

func appendCodecPipeline(prev *avframe.Pipeline, srcMap map[avframe.CodecType]*avframe.Pipeline, targetCodecType avframe.CodecType, createPlugin func() (avframe.Processor, error)) (pipeline *avframe.Pipeline, backout func(*error), err error) {
	if pipeline, found := srcMap[targetCodecType]; found {
		return pipeline, func(err *error) {}, nil
	}

	if processor, e := createPlugin(); e != nil {
		return nil, func(err *error) {}, e
	} else {
		pipeline = avframe.NewPipeline(processor)
	}

	srcMap[targetCodecType] = pipeline
	prev.AddNext(pipeline, avframe.WithoutPayloadType(avframe.PayloadTypeVideo))

	return pipeline, func(err *error) {
		if err != nil && *err != nil {
			prev.RemoveNext(pipeline)
			delete(srcMap, targetCodecType)
			pipeline.Close()
		}
	}, nil
}

// Unified method to add decoder and encoder if needed
func (s *Stream) appendTranscodePipeline(ctx context.Context, prevPipeline *avframe.Pipeline, targetCodecType avframe.CodecType) (encoderPipeline *avframe.Pipeline, backouts []func(*error), err error) {
	metadata := prevPipeline.Metadata()
	isAudio := targetCodecType.IsAudio()
	var originalCodecType avframe.CodecType
	var decoderPipelineMap map[avframe.CodecType]*avframe.Pipeline
	var encoderPipelineMap map[avframe.CodecType]*avframe.Pipeline
	if isAudio {
		originalCodecType = metadata.AudioCodecType
		decoderPipelineMap = s.audioDecoders
		encoderPipelineMap = s.audioEncoders
	} else {
		originalCodecType = metadata.VideoCodecType
		decoderPipelineMap = s.videoDecoders
		encoderPipelineMap = s.videoEncoders
	}

	decoderPipeline, decoderBackout, err := appendCodecPipeline(prevPipeline, decoderPipelineMap, originalCodecType, func() (avframe.Processor, error) {
		return plugin.CreateDecoderPlugin(ctx, originalCodecType, metadata)
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "create decoder")
	}

	backouts = append(backouts, decoderBackout)

	var encoderBackout func(*error)
	encoderPipeline, encoderBackout, err = appendCodecPipeline(decoderPipeline, encoderPipelineMap, targetCodecType, func() (avframe.Processor, error) {
		return plugin.CreateEncoderPlugin(ctx, targetCodecType, decoderPipeline.Metadata())
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "create encoder")
	}

	backouts = append(backouts, encoderBackout)

	return encoderPipeline, backouts, nil
}

func (s *Stream) appendDemuxer(prev *avframe.Pipeline, metadata avframe.Metadata) (pipeline *avframe.Pipeline, backout func(*error), err error) {
	if s.demuxer != nil {
		return s.demuxer, func(err *error) {}, nil
	}

	if processor, e := plugin.CreateDemuxerPlugin(s.ctx, metadata); e != nil {
		return nil, func(err *error) {}, e
	} else {
		pipeline = avframe.NewPipeline(processor)
	}

	if e := prev.AddNext(pipeline, avframe.WithAllPayloadTypes()); e != nil {
		return nil, func(err *error) {}, e
	}

	s.demuxer = pipeline

	return pipeline, func(err *error) {
		if err != nil && *err != nil {
			prev.RemoveNext(pipeline)
			s.demuxer = nil
			pipeline.Close()
		}
	}, nil
}

func (s *Stream) appendMuxer(fmtType avframe.FmtType, metadata avframe.Metadata) (pipeline *avframe.Pipeline, backout func(*error), err error) {
	if s.muxers[fmtType] != nil {
		return s.muxers[fmtType], func(err *error) {}, nil
	}

	if processor, e := plugin.CreateMuxerPlugin(s.ctx, fmtType, metadata); e != nil {
		return nil, func(err *error) {}, e
	} else {
		pipeline = avframe.NewPipeline(processor)
	}

	audioEncoderPipeline, found := s.audioEncoders[metadata.AudioCodecType]
	if found {
		if e := audioEncoderPipeline.AddNext(pipeline, avframe.WithoutPayloadType(avframe.PayloadTypeVideo)); e != nil {
			return nil, func(err *error) {}, e
		}
	} else {
		if e := s.demuxer.AddNext(pipeline, avframe.WithoutPayloadType(avframe.PayloadTypeVideo)); e != nil {
			return nil, func(err *error) {}, e
		}
	}

	videoEncoderPipeline, found := s.videoEncoders[metadata.VideoCodecType]
	if found {
		if e := videoEncoderPipeline.AddNext(pipeline, avframe.WithoutPayloadType(avframe.PayloadTypeAudio)); e != nil {
			return nil, func(err *error) {}, e
		}
	} else {
		if e := s.demuxer.AddNext(pipeline, avframe.WithoutPayloadType(avframe.PayloadTypeAudio)); e != nil {
			return nil, func(err *error) {}, e
		}
	}

	s.muxers[fmtType] = pipeline

	return pipeline, func(err *error) {
		if err != nil && *err != nil {
			if audioEncoderPipeline != nil {
				audioEncoderPipeline.RemoveNext(pipeline)
			}
			if videoEncoderPipeline != nil {
				videoEncoderPipeline.RemoveNext(pipeline)
			}
			s.demuxer.RemoveNext(pipeline)
			s.muxers[fmtType] = nil
			pipeline.Close()
		}
	}, nil
}

func (s *Stream) createSubStream(subFmtType avframe.FmtType, fmtSupported avframe.FmtSupported) (ss *subStream, errResult error) {
	errResult = errors.New("failed to create substream")

	if s.noopProcessor == nil {
		return nil, fmt.Errorf("stream not activated")
	}

	var allBackouts []func(*error)

	create := func(processor *avframe.Pipeline) (ss *subStream, err error) {
		ss, errResult = newSubStream(s.ctx, processor,
			WithOnSubscriberEmpty(func(sub peer.Subscriber) {
				ss.SetIdleTimeout(time.Now().Add(time.Second * s.settings.IdleTimeout))
			}),
			WithLogger(s.logger.WithField("substream", processor.Format())))

		go func(ss *subStream) {
			<-ss.Done()
			s.lock.Lock()
			defer s.lock.Unlock()
			if s.subStreams[ss.Format()] == ss {
				delete(s.subStreams, ss.Format())
			}
		}(ss)
		return
	}

	defer func() {
		for _, backout := range allBackouts {
			backout(&errResult)
		}
	}()

	// if source fmt type is the same as sub fmt type, then use noop processor
	sourceFmtType := s.publisher.Metadata().FmtType
	if sourceFmtType == subFmtType {
		return create(s.noopProcessor)
	}

	// if demuxer already exists, then use it
	demuxerPipeline, demuxerBackout, err := s.appendDemuxer(s.noopProcessor, s.publisher.Metadata())
	if err != nil {
		return nil, errors.Wrap(err, "append demuxer")
	}

	allBackouts = append(allBackouts, demuxerBackout)

	sourceAudioCodecType := s.publisher.Metadata().AudioCodecType
	sourceVideoCodecType := s.publisher.Metadata().VideoCodecType

	// get suitable codec type for target fmt type
	var targetAudioCodecType avframe.CodecType
	var targetVideoCodecType avframe.CodecType
	if fmtSupported != nil {
		targetAudioCodecType = fmtSupported.GetSuitableAudioCodecType(subFmtType, sourceAudioCodecType)
		targetVideoCodecType = fmtSupported.GetSuitableVideoCodecType(subFmtType, sourceVideoCodecType)
	} else {
		targetAudioCodecType = avframe.GetSuitableAudioCodecType(subFmtType, sourceAudioCodecType)
		targetVideoCodecType = avframe.GetSuitableVideoCodecType(subFmtType, sourceVideoCodecType)
	}

	if targetAudioCodecType == avframe.CodecTypeUnknown || targetVideoCodecType == avframe.CodecTypeUnknown {
		return nil, fmt.Errorf("no suitable codec type for target fmt type[%s]", subFmtType)
	}

	// add audio decoder and encoder if needed
	if targetAudioCodecType != sourceAudioCodecType {
		if _, backouts, err := s.appendTranscodePipeline(s.ctx,
			demuxerPipeline,
			targetAudioCodecType); err != nil {
			return nil, errors.Wrap(err, "add audio codec pipeline")
		} else {
			allBackouts = append(allBackouts, backouts...)
		}
	}

	// add video decoder and encoder if needed
	if targetVideoCodecType != sourceVideoCodecType {
		if _, backouts, err := s.appendTranscodePipeline(s.ctx,
			demuxerPipeline,
			targetVideoCodecType); err != nil {
			return nil, errors.Wrap(err, "add video codec pipeline")
		} else {
			allBackouts = append(allBackouts, backouts...)
		}
	}

	// create muxer metadata
	muxerMetadata := demuxerPipeline.Metadata()
	muxerMetadata.FmtType = subFmtType
	muxerMetadata.AudioCodecType = targetAudioCodecType
	muxerMetadata.VideoCodecType = targetVideoCodecType

	muxerPipeline, muxerBackout, err := s.appendMuxer(subFmtType, muxerMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "append muxer")
	}

	allBackouts = append(allBackouts, muxerBackout)

	return create(muxerPipeline)

}

func (s *Stream) publisherSet() bool {
	return atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.publisher))) != nil
}

func (s *Stream) getOrCreateSubStream(fmtType avframe.FmtType, fmtSupported avframe.FmtSupported) (*subStream, error) {
	if _, ok := s.subStreams[fmtType]; !ok {
		subStream, err := s.createSubStream(fmtType, fmtSupported)
		if err != nil {
			return nil, errors.Wrap(err, "new substream")
		}

		s.subStreams[fmtType] = subStream
	}

	return s.subStreams[fmtType], nil
}

func (s *Stream) GetSubStream(fmtType avframe.FmtType) *subStream {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.subStreams[fmtType]
}

func (s *Stream) GetSubStreams() []*subStream {
	s.lock.RLock()
	defer s.lock.RUnlock()
	subStreams := make([]*subStream, 0, len(s.subStreams))
	for _, subStream := range s.subStreams {
		subStreams = append(subStreams, subStream)
	}
	return subStreams
}

func (s *Stream) Subscribe(sub peer.Subscriber, onResult SubscribeResultFunc) (processor avframe.Processor, err error) {
	fmtType := sub.Format()
	subVideoCodecSupported := sub.VideoCodecSupported()
	subAudioCodecSupported := sub.AudioCodecSupported()
	fmtSupported := avframe.DupFmtSupported()

	// Retrieve the current format settings
	currentFmt := fmtSupported[fmtType]

	if subVideoCodecSupported != nil {
		currentFmt.VideoCodecTypes = subVideoCodecSupported
	}
	if subAudioCodecSupported != nil {
		currentFmt.AudioCodecTypes = subAudioCodecSupported
	}

	// Put the modified struct back into the map
	fmtSupported[fmtType] = currentFmt

	s.lock.Lock()
	defer s.lock.Unlock()

	if atomic.LoadUint32(&s.closed) == 1 {
		return nil, errcode.New(errcode.ErrStreamClosed, nil)
	}

	if !s.publisherSet() {
		s.waitingSubs = append(s.waitingSubs, waitingSub{
			sub:      sub,
			onResult: onResult,
		})

		return nil, errcode.New(errcode.ErrPublisherNotSet, nil)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}

		if onResult != nil {
			onResult(sub, processor, err)
		}
	}()

	subStream, err := s.getOrCreateSubStream(fmtType, fmtSupported)
	if err != nil {
		return nil, errors.Wrap(err, "get or create substream")
	}

	err = subStream.Subscribe(sub)
	if err != nil {
		return nil, errors.Wrap(err, "subscribe substream")
	}

	return subStream, nil
}

func (s *Stream) Publish(publisher peer.Publisher) error {
	s.logger.WithField("publisher", publisher).Info("publish")

	s.lock.Lock()
	defer s.lock.Unlock()

	if atomic.LoadUint32(&s.closed) == 1 {
		return errcode.New(errcode.ErrStreamClosed, nil)
	}

	if publisher == nil {
		return errcode.New(errcode.ErrPublisherIsNil, nil)
	}

	if s.publisher != nil {
		return errcode.New(errcode.ErrPublisherAlreadySet, nil)
	}

	s.publisher = publisher

	if s.noopProcessor == nil {
		s.noopProcessor = avframe.NewPipeline(avframe.NewNoopProcessor(s.publisher.Metadata()))
	}

	subs := []waitingSub{}
	subs = append(subs, s.waitingSubs...)
	s.waitingSubs = []waitingSub{}

	go func() {
		for _, sub := range subs {
			_, err := s.Subscribe(sub.sub, sub.onResult)
			if err != nil {
				s.logger.WithFields(map[string]interface{}{
					"sub": sub.sub,
				}).Error(err)
			}
		}
	}()

	s.chWaitPublisher <- struct{}{}

	return nil
}

func (s *Stream) Unpublish(publisher peer.Publisher) error {
	s.logger.Info("unpublish")

	s.lock.Lock()
	defer s.lock.Unlock()

	if atomic.LoadUint32(&s.closed) == 1 {
		return errcode.New(errcode.ErrStreamClosed, nil)
	}

	if publisher != s.publisher {
		return errcode.New(errcode.ErrPublisherNotMatch, nil)
	}

	if s.publisher == nil {
		return errcode.New(errcode.ErrPublisherNotSet, nil)
	}

	s.publisher = nil

	return nil
}

func (s *Stream) Unsubscribe(sub peer.Subscriber) error {
	s.logger.WithField("sub", sub).Info("unsubscribe")

	fmtType := sub.Format()
	subStream := s.GetSubStream(fmtType)
	if subStream == nil {
		return errcode.New(errcode.ErrSubStreamNotExists, nil)
	}
	subStream.Unsubscribe(sub)

	return nil
}

func (s *Stream) destory() {
	s.logger.Info("stream destory")

	if !atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.demuxer != nil {
		s.demuxer.Close()
	}
	if s.noopProcessor != nil {
		s.noopProcessor.Close()
	}

	for _, subStream := range s.subStreams {
		subStream.Close()
	}

	for _, decoder := range s.audioDecoders {
		decoder.Close()
	}
	for _, decoder := range s.videoDecoders {
		decoder.Close()
	}
	for _, encoder := range s.audioEncoders {
		encoder.Close()
	}
	for _, encoder := range s.videoEncoders {
		encoder.Close()
	}
	for _, muxer := range s.muxers {
		muxer.Close()
	}

	close(s.chWaitPublisher)
	close(s.chPublisherSwapped)
}

func (s *Stream) processPublisherStream(publisher peer.Publisher) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("process publisher stream panic: %v", r)
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return nil
		case <-s.chPublisherSwapped:
			s.logger.Info("swapped publisher")
			return nil
		default:
		}

		frame, err := publisher.Read()
		if err != nil {
			if err != io.EOF {
				s.logger.WithError(err).Error("read publisher")
			}
			return err
		}

		if err := s.noopProcessor.Write(frame); err != nil {
			s.logger.WithError(err).Error("write noop processor")
		}
	}
}

func (s *Stream) getCurrentPublisher() peer.Publisher {
	s.lock.RLock()
	publisher := s.publisher
	s.lock.RUnlock()
	return publisher
}

func (s *Stream) run() {
	defer func() {
		if r := recover(); r != nil {
			s.logger.WithError(fmt.Errorf("panic: %v", r)).Error("stream run panic")
		}

		if s.ctx.Err() == nil {
			s.cancel()
		}

		s.destory()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		publisher := s.getCurrentPublisher()

		if publisher == nil {
			select {
			case <-s.ctx.Done():
				return
			case <-s.chWaitPublisher:
				publisher = s.getCurrentPublisher()
				if publisher == nil {
					panic("publisher is nil")
				}
			}
		}

		if err := s.processPublisherStream(publisher); err != nil {
			s.logger.WithError(err).Error("process publisher stream")
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *Stream) Close() error {
	s.logger.Info("stream close")

	s.cancel()

	return nil
}

func (s *Stream) Done() <-chan struct{} {
	return s.ctx.Done()
}
