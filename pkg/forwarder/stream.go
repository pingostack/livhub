package forwarder

import (
	"context"
	"sync"

	streaminterceptor "github.com/pingostack/neon/pkg/stream_interceptor"
	"github.com/sirupsen/logrus"
)

type Stream struct {
	id              string
	medias          map[streaminterceptor.MediaKind]*StreamMedia
	desc            *StreamDescription
	logger          *logrus.Entry
	lock            sync.RWMutex
	waitMediaCtx    context.Context
	waitMediaCancel context.CancelFunc
	ctx             context.Context
}

func NewStream(ctx context.Context, desc *StreamDescription, logger *logrus.Entry) *Stream {
	s := &Stream{
		desc:   desc,
		medias: make(map[streaminterceptor.MediaKind]*StreamMedia),
		logger: logger,
		id:     desc.ID,
		ctx:    ctx,
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

	if _, ok := s.medias[meta.Kind]; ok {
		return
	}

	s.medias[meta.Kind] = NewStreamMedia(meta, s.logger)
	s.desc.Medias[meta.Kind] = meta

	if s.desc.HasAudio && s.desc.HasVideo {
		if len(s.medias) >= 2 &&
			s.medias[streaminterceptor.MediaTypeAudio] != nil &&
			s.medias[streaminterceptor.MediaTypeVideo] != nil {
			s.waitMediaCancel()
		}
	} else if s.desc.HasAudio && meta.Kind == streaminterceptor.MediaTypeAudio {
		s.waitMediaCancel()
	} else if s.desc.HasVideo && meta.Kind == streaminterceptor.MediaTypeVideo {
		s.waitMediaCancel()
	}
}

func (s *Stream) Media(mediaType streaminterceptor.MediaKind) *StreamMedia {
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
