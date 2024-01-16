package forwarder

import (
	"sync"

	streaminterceptor "github.com/pingostack/neon/pkg/stream_interceptor"
	"github.com/sirupsen/logrus"
)

type StreamMedia struct {
	formats  map[string]*StreamFormat
	lock     sync.RWMutex
	logger   *logrus.Entry
	metadata *streaminterceptor.Metadata
}

func NewStreamMedia(metadata *streaminterceptor.Metadata, logger *logrus.Entry) *StreamMedia {
	m := &StreamMedia{
		metadata: metadata,
		formats:  make(map[string]*StreamFormat),
		logger:   logger.WithField("MediaKind", metadata.Kind),
	}

	return m
}

func (m *StreamMedia) Format(name string) *StreamFormat {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.formats[name]
}

func (m *StreamMedia) AddFormat(format *StreamFormat) {
	m.logger.Infof("StreamMedia.AddFormat(%v)", format)
	m.lock.Lock()
	defer m.lock.Unlock()

	m.formats[format.Name()] = format
}

func (m *StreamMedia) Metadata() *streaminterceptor.Metadata {
	return m.metadata
}
