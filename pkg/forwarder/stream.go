package forwarder

import (
	"context"
	"sync"

	streaminterceptor "github.com/pingostack/neon/pkg/stream_interceptor"
	"github.com/sirupsen/logrus"
)

type Stream struct {
	id              string
	medias          map[streaminterceptor.MediaType]*StreamMedia
	desc            *StreamDescription
	logger          *logrus.Entry
	lock            sync.RWMutex
	waitMediaCtx    context.Context
	waitMediaCancel context.CancelFunc
}

func NewStream(ctx context.Context, desc *StreamDescription, logger *logrus.Entry) *Stream {
	s := &Stream{
		desc:   desc,
		medias: make(map[streaminterceptor.MediaType]*StreamMedia),
		logger: logger,
		id:     desc.ID,
	}

	s.waitMediaCtx, s.waitMediaCancel = context.WithCancel(ctx)

	return s
}

func (s *Stream) ID() string {
	return s.id
}

func (s *Stream) AddMedia(meta *streaminterceptor.Metadata) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.medias[meta.MediaType]; ok {
		return
	}

	s.medias[meta.MediaType] = NewStreamMedia(meta, s.logger)
	s.desc.Medias = append(s.desc.Medias, meta)

	if s.desc.HasAudio && s.desc.HasVideo {
		if len(s.medias) >= 2 &&
			s.medias[streaminterceptor.MediaTypeAudio] != nil &&
			s.medias[streaminterceptor.MediaTypeVideo] != nil {
			s.waitMediaCancel()
		}
	} else if s.desc.HasAudio && meta.MediaType == streaminterceptor.MediaTypeAudio {
		s.waitMediaCancel()
	} else if s.desc.HasVideo && meta.MediaType == streaminterceptor.MediaTypeVideo {
		s.waitMediaCancel()
	}
}

func (s *Stream) Media(mediaType streaminterceptor.MediaType) *StreamMedia {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, ok := s.medias[mediaType]; !ok {
		return nil
	}

	return s.medias[mediaType]
}

func (s *Stream) Medias() []*StreamMedia {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if len(s.medias) == 0 {
		return nil
	}

	medias := make([]*StreamMedia, 0)
	for _, media := range s.medias {
		medias = append(medias, media)
	}

	return medias
}

func (s *Stream) GatheringMedias(ctx context.Context) []*StreamMedia {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.waitMediaCtx.Done():
			return s.Medias()
		}
	}
}
