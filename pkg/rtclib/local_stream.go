package rtclib

import (
	"context"

	"github.com/pingostack/livhub/pkg/deliver"
	"github.com/pingostack/livhub/pkg/eventemitter"
	"github.com/pingostack/livhub/pkg/logger"
	"github.com/pingostack/livhub/pkg/rtclib/transport"
	"github.com/pkg/errors"
)

type LocalStream struct {
	*transport.Transport
	ctx          context.Context
	cancel       context.CancelFunc
	logger       logger.Logger
	eventemitter eventemitter.EventEmitter
}

func NewLocalStream(transport *transport.Transport) (*LocalStream, error) {
	ls := &LocalStream{
		Transport:    transport,
		logger:       transport.Logger(),
		eventemitter: eventemitter.NewEventEmitter(transport.Context(), defaultEventEmitterLength, transport.Logger()),
	}

	ls.ctx, ls.cancel = context.WithCancel(transport.Context())

	if err := ls.validate(); err != nil {
		return nil, errors.Wrap(err, "invalid local stream")
	}

	return ls, nil
}

func (ls *LocalStream) validate() error {
	if ls.Transport == nil {
		return errors.New("transport not set")
	}

	if ls.ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		ls.ctx = ctx
		ls.cancel = cancel
	}

	if ls.logger == nil {
		ls.logger = logger.DefaultLogger
	}

	return nil
}

func (ls *LocalStream) AddTrack(codec deliver.CodecType, clockRate uint32, logger logger.Logger) (track *TrackLocl, err error) {
	return NewTrackLocl(ls.ctx, codec, clockRate, ls.Transport.AddTrack, logger)
}

func (ls *LocalStream) Close() {
	ls.cancel()
	ls.Transport.Close()
}
