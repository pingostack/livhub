package rtclib

import (
	"context"
	"time"

	"github.com/pingostack/neon/pkg/logger"
	"github.com/pingostack/neon/protocols/rtclib/transport"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/errors"
)

type RemoteStream struct {
	*transport.Transport
	ctx     context.Context
	cancel  context.CancelFunc
	logger  logger.Logger
	chTrack chan *TrackRemote
}

func NewRemoteStream(transport *transport.Transport) (*RemoteStream, error) {
	p := &RemoteStream{
		Transport: transport,
		logger:    transport.Logger(),
		chTrack:   make(chan *TrackRemote, 2),
	}

	p.ctx, p.cancel = context.WithCancel(transport.Context())

	if err := p.validate(); err != nil {
		return nil, errors.Wrap(err, "invalid remote stream")
	}

	p.Transport.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		p.logger.Infof("got track %s(%s)", track.ID(), track.Kind())
		p.chTrack <- NewTrackRemote(p.ctx, track, receiver, p.Transport.WriteRTCP, p.logger)
	})

	return p, nil
}

func (p *RemoteStream) validate() error {
	if p.Transport == nil {
		return errors.New("transport not set")
	}

	if p.ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		p.ctx = ctx
		p.cancel = cancel
	}

	if p.logger == nil {
		p.logger = logger.DefaultLogger
	}

	return nil
}

func (p *RemoteStream) GatheringTracks(gatherAudioTrack, gatherVideoTrack bool, timeout time.Duration) ([]*TrackRemote, error) {
	gotAudio, gotVideo := false, false

	tracks := make([]*TrackRemote, 0)
	for {
		select {
		case <-time.After(timeout):
			return nil, errors.New("gather tracks timeout")
		case <-p.ctx.Done():
			return nil, errors.New("gather tracks canceled")
		case track := <-p.chTrack:
			tracks = append(tracks, track)
			if track.IsAudio() {
				gotAudio = true
			}

			if track.IsVideo() {
				gotVideo = true
			}

			if gatherAudioTrack && gatherVideoTrack {
				if gotAudio && gotVideo {
					return tracks, nil
				}
			} else if gatherAudioTrack {
				if gotAudio {
					return tracks, nil
				}
			} else if gatherVideoTrack {
				if gotVideo {
					return tracks, nil
				}
			} else {
				return tracks, nil
			}
		}
	}
}
