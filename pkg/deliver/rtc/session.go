package rtc

import (
	"context"
	"time"

	"github.com/pingostack/livhub/internal/core"
	"github.com/pingostack/livhub/internal/core/router"
	"github.com/pingostack/livhub/pkg/rtclib"
	"github.com/pingostack/livhub/pkg/rtclib/sdpassistor"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ServSession struct {
	router.Session
	pm     router.PeerParams
	ctx    context.Context
	logger *logrus.Entry
	dest   *FrameDestination
	src    *FrameSource
	sf     rtclib.StreamFactory
}

func NewServSession(ctx context.Context, sf rtclib.StreamFactory, pm router.PeerParams, logger *logrus.Entry) *ServSession {
	s := &ServSession{
		ctx: ctx,
		pm:  pm,
		sf:  sf,
		logger: logger.WithFields(logrus.Fields{
			"session-type": "serv-session",
		}),
	}

	return s
}

func (s *ServSession) Publish(keyFrameInterval time.Duration, sdpOffer string) (*webrtc.SessionDescription, error) {
	logger := s.logger

	src, err := NewFrameSource(s.ctx, s.sf, false, keyFrameInterval, logger)
	if err != nil {
		logger.WithError(err).Error("failed to create frame source")
		return nil, errors.Wrap(err, "failed to create frame source")
	}

	err = src.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdpOffer,
	})
	if err != nil {
		logger.WithError(err).Error("failed to set remote description")
		return nil, errors.Wrap(err, "failed to set remote description")
	}

	_, err = src.CreateAnswer(nil)
	if err != nil {
		logger.WithError(err).Error("failed to create answer")
		return nil, errors.Wrap(err, "failed to create answer")
	}

	err = src.Start()
	if err != nil {
		logger.WithError(err).Error("failed to start frame source")
		return nil, errors.Wrap(err, "failed to start frame source")
	}

	logger.WithField("metadata", src.Metadata().String()).Debug("frame source metadata")

	// create session
	s.pm.Producer = true
	s.pm.HasAudio = src.Metadata().HasAudio()
	s.pm.HasVideo = src.Metadata().HasVideo()
	s.pm.HasDataChannel = src.Metadata().HasData()

	s.Session = core.NewSession(s.ctx, s.pm, logger)
	session := s.Session

	err = session.BindFrameSource(src)
	if err != nil {
		logger.WithError(err).Error("failed to bind frame source")
		return nil, errors.Wrap(err, "failed to bind frame source")
	}

	err = session.Join()
	if err != nil {
		logger.WithError(err).Error("join failed")
		return nil, errors.Wrap(err, "join failed")
	}

	lsdp, err := src.GatheringCompleteLocalSdp(context.Background())
	if err != nil {
		logger.WithError(err).Error("failed to get completed sdp")
		return nil, errors.Wrap(err, "failed to get completed sdp")
	}

	s.src = src

	return &lsdp, nil
}

func (s *ServSession) Subscribe(sdpOffer string, timeout time.Duration) (*webrtc.SessionDescription, error) {
	logger := s.logger
	hasAudio, hasVideo, hasData, err := sdpassistor.GetPayloadStatus(sdpOffer, webrtc.SDPTypeOffer)
	if err != nil {
		logger.WithError(err).Error("failed to get payload status")
		return nil, errors.Wrap(err, "failed to get payload status")
	}

	s.pm.Producer = false
	s.pm.HasAudio = hasAudio
	s.pm.HasVideo = hasVideo
	s.pm.HasDataChannel = hasData

	s.Session = core.NewSession(s.ctx, s.pm, logger)

	dest, err := NewFrameDestination(s.ctx, s.sf,
		false, logger)
	if err != nil {
		logger.WithError(err).Error("failed to create frame destination")
		return nil, errors.Wrap(err, "failed create frame destination")
	}

	err = dest.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdpOffer,
	})
	if err != nil {
		logger.WithError(err).Error("failed to set remote description")
		return nil, errors.Wrap(err, "failed to set remote description")
	}

	err = s.BindFrameDestination(dest)
	if err != nil {
		logger.WithError(err).Error("failed to bind frame destination")
		return nil, errors.Wrap(err, "failed to bind frame source")
	}

	err = s.Join()
	if err != nil {
		if errors.Is(err, router.ErrPaddingDestination) {
			select {
			case <-s.ctx.Done():
				return nil, errors.Wrap(err, "context done")
			case err = <-dest.SourceCompletePromise():
				if err != nil {
					logger.WithError(err).Error("join failed")
					return nil, errors.Wrap(err, "join failed")
				} else {
					logger.Info("join success")
				}
			case <-time.After(timeout):
				logger.WithField("timeout", timeout).Error("join timeout")
				return nil, errors.New("join timeout")
			}
		} else {
			logger.WithError(err).Error("join failed")
			return nil, errors.Wrap(err, "join failed")
		}
	}

	_, err = dest.CreateAnswer(nil)
	if err != nil {
		logger.WithError(err).Error("failed to create answer")
		return nil, errors.Wrap(err, "failed to create answer")
	}

	lsdp, err := dest.GatheringCompleteLocalSdp(context.Background())
	if err != nil {
		logger.WithError(err).Error("failed to get completed sdp")
		return nil, errors.Wrap(err, "failed to get completed sdp")
	}

	err = dest.Start()
	if err != nil {
		logger.WithError(err).Error("failed to start frame destination")
		return nil, errors.Wrap(err, "failed to start frame destination")
	}

	s.dest = dest

	return &lsdp, nil
}
