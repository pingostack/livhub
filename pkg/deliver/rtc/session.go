package rtc

import (
	"context"
	"time"

	"github.com/pingostack/neon/internal/core/router"
	"github.com/pingostack/neon/pkg/rtclib"
	"github.com/pingostack/neon/pkg/rtclib/sdpassistor"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ServSession struct {
	router.Session
	ctx        context.Context
	logger     *logrus.Entry
	dest       *FrameDestination
	src        *FrameSource
	sf         rtclib.StreamFactory
	newSession func(hasAudio bool, hasVideo bool, hasData bool) router.Session
}

func NewServSession(ctx context.Context, sf rtclib.StreamFactory, logger *logrus.Entry, f func(hasAudio bool, hasVideo bool, hasData bool) router.Session) *ServSession {
	servSession := &ServSession{
		ctx: ctx,
		sf:  sf,
		logger: logger.WithFields(logrus.Fields{
			"session-type": "servSession-session",
		}),
		newSession: f,
	}

	return servSession
}

func (servSession *ServSession) Publish(keyFrameInterval time.Duration, sdpOffer string) (*webrtc.SessionDescription, error) {
	logger := servSession.logger

	src, err := NewFrameSource(servSession.ctx, servSession.sf, false, keyFrameInterval, logger)
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
	servSession.Session = servSession.newSession(src.Metadata().HasAudio(), src.Metadata().HasVideo(), src.Metadata().HasData())
	session := servSession.Session

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

	servSession.src = src

	return &lsdp, nil
}

func (servSession *ServSession) Subscribe(sdpOffer string, timeout time.Duration) (*webrtc.SessionDescription, error) {
	logger := servSession.logger
	hasAudio, hasVideo, hasData, err := sdpassistor.GetPayloadStatus(sdpOffer, webrtc.SDPTypeOffer)
	if err != nil {
		logger.WithError(err).Error("failed to get payload status")
		return nil, errors.Wrap(err, "failed to get payload status")
	}

	servSession.Session = servSession.newSession(hasAudio, hasVideo, hasData)

	dest, err := NewFrameDestination(servSession.ctx, servSession.sf,
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

	err = servSession.BindFrameDestination(dest)
	if err != nil {
		logger.WithError(err).Error("failed to bind frame destination")
		return nil, errors.Wrap(err, "failed to bind frame source")
	}

	err = servSession.Join()
	if err != nil {
		if errors.Is(err, router.ErrPaddingDestination) {
			select {
			case <-servSession.Context().Done():
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

	servSession.dest = dest

	return &lsdp, nil
}
