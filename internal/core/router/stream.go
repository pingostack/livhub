package router

import (
	"context"
	"sync"

	sourcemanager "github.com/pingostack/neon/internal/core/router/source_manager"
	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Stream interface {
	GetFormat(fmtName string) (StreamFormat, error)
	AddFrameSource(source deliver.FrameSource) error
	AddFrameDestination(dest deliver.FrameDestination) (err error)
	Close()
}

type StreamImpl struct {
	ctx     context.Context
	cancel  context.CancelFunc
	formats map[string]StreamFormat
	lock    sync.RWMutex
	closed  bool
	//pendingDests []deliver.FrameDestination
	logger       *logrus.Entry
	sm           *sourcemanager.Instance
	paddingDests []deliver.FrameDestination
}

func NewStreamImpl(ctx context.Context, id string) Stream {
	s := &StreamImpl{
		formats: make(map[string]StreamFormat),
		logger:  logrus.WithField("stream", id),
		sm:      sourcemanager.NewInstance(),
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	go func() {
		<-ctx.Done()

		s.lock.Lock()
		defer func() {
			if err := recover(); err != nil {
				s.logger.WithField("error", err).Error("StreamImpl goroutine panic")
			}
			s.lock.Unlock()
		}()

		s.closed = true

		for _, f := range s.formats {
			f.Close()
		}
	}()

	return s
}

func (s *StreamImpl) GetFormat(fmtName string) (StreamFormat, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.closed {
		return nil, ErrStreamClosed
	}

	format, ok := s.formats[fmtName]
	if !ok {
		return nil, ErrStreamFormatNotFound
	}

	return format, nil
}

func (s *StreamImpl) AddFrameSource(source deliver.FrameSource) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	if !s.sm.AddIfNotExist(source) {
		return ErrFrameSourceExists
	}

	for _, dest := range s.paddingDests {
		s.addFrameDestination(dest)
	}

	return nil
}

func (s *StreamImpl) addFrameDestination(dest deliver.FrameDestination) (err error) {
	fmtName := dest.Metadata().FormatName()
	format, ok := s.formats[fmtName]
	if !ok {
		format, err = NewStreamFormat(s.ctx, dest.FormatSettings(), WithFrameSourceManager(s.sm))
		if err != nil {
			return errors.Wrap(err, "failed to create stream format")
		}

		s.formats[fmtName] = format
	}

	format.AddDestination(dest)

	return nil
}

func (s *StreamImpl) AddFrameDestination(dest deliver.FrameDestination) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	if s.sm.DefaultSource() == nil {
		s.paddingDests = append(s.paddingDests, dest)
		return ErrPaddingDestination
	}

	return s.addFrameDestination(dest)
}

func (s *StreamImpl) Close() {
	s.cancel()
}
