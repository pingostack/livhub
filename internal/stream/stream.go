package stream

import (
	"context"
	"sync"
	"time"

	streaminterceptor "github.com/pingostack/neon/pkg/stream_interceptor"
	"github.com/sirupsen/logrus"
)

type Stream struct {
	id             string
	medias         map[streaminterceptor.MediaType]*StreamMedia
	desc           *StreamDescription
	logger         *logrus.Entry
	lock           sync.RWMutex
	chGatherMedias chan *StreamMedia
}

func NewStream(desc *StreamDescription, logger *logrus.Entry) *Stream {
	s := &Stream{
		desc:           desc,
		medias:         make(map[streaminterceptor.MediaType]*StreamMedia),
		logger:         logger,
		id:             desc.ID,
		chGatherMedias: make(chan *StreamMedia, 2),
	}

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

	s.chGatherMedias <- s.medias[meta.MediaType]
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

	medias := make([]*StreamMedia, 0)
	for _, media := range s.medias {
		medias = append(medias, media)
	}

	return medias
}

func (s *Stream) GatherMedias(ctx context.Context) []*StreamMedia {
	gotAudio, gotVideo := false, false

	medias := make([]*StreamMedia, 0)
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-time.After(s.desc.SyncTimeout):
			s.logger.Warnf("gather medias timeout")
			if len(medias) == 0 {
				return nil
			} else {
				return medias
			}

		case media := <-s.chGatherMedias:
			s.logger.Infof("gather media %v", media.Metadata())
			medias = append(medias, media)
			if s.desc.HasAudio && s.desc.HasVideo {
				if gotAudio && gotVideo {
					return medias
				}
			} else if s.desc.HasAudio {
				if gotAudio {
					return medias
				}
			} else if s.desc.HasVideo {
				if gotVideo {
					return medias
				}
			} else {
				return nil
			}
		}
	}
}
