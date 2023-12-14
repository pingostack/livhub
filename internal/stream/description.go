package stream

import (
	"time"

	streaminterceptor "github.com/pingostack/neon/pkg/stream_interceptor"
)

type StreamDescription struct {
	HasVideo    bool
	HasAudio    bool
	SyncTimeout time.Duration
	ID          string
	Medias      []*streaminterceptor.Metadata
}
