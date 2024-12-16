package rtclib

import (
	"context"
	"time"

	"github.com/pingostack/livhub/pkg/logger"
	"github.com/pingostack/livhub/pkg/rtclib/transport"
	"github.com/pion/webrtc/v4"
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
	rs := &RemoteStream{
		Transport: transport,
		logger:    transport.Logger(),
		chTrack:   make(chan *TrackRemote, 2),
	}

	rs.ctx, rs.cancel = context.WithCancel(transport.Context())

	if err := rs.validate(); err != nil {
		return nil, errors.Wrap(err, "invalid remote stream")
	}

	rs.Transport.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		rs.logger.Infof("got track %s(%s)", track.ID(), track.Kind())
		rs.chTrack <- NewTrackRemote(rs.ctx, track, receiver, rs.Transport.WriteRTCP, rs.logger)
	})

	return rs, nil
}

func (rs *RemoteStream) validate() error {
	if rs.Transport == nil {
		return errors.New("transport not set")
	}

	if rs.ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		rs.ctx = ctx
		rs.cancel = cancel
	}

	if rs.logger == nil {
		rs.logger = logger.DefaultLogger
	}

	return nil
}

func (rs *RemoteStream) GatheringTracks(gatherAudioTrack, gatherVideoTrack bool, timeout time.Duration) ([]*TrackRemote, error) {
	gotAudio, gotVideo := false, false

	tracks := make([]*TrackRemote, 0)
	for {
		select {
		case <-time.After(timeout):
			return nil, errors.New("gather tracks timeout")
		case <-rs.ctx.Done():
			return nil, errors.New("gather tracks canceled")
		case track := <-rs.chTrack:
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

func (rs *RemoteStream) Close() {
	rs.cancel()
	rs.Transport.Close()
}
