package pms

import (
	"context"
	"time"

	"github.com/pingostack/neon/internal/rtc"
	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pingostack/neon/protocols/rtclib"
	"github.com/pion/webrtc/v4"
	"github.com/sirupsen/logrus"
)

type FrameSource struct {
	deliver.FrameSource
	ctx          context.Context
	cancel       context.CancelFunc
	remoteStream *rtclib.RemoteStream
	logger       *logrus.Entry
	offer        webrtc.SessionDescription
	answer       webrtc.SessionDescription
	metadata     Metadata
}

func NewFrameSource(ctx context.Context, offer webrtc.SessionDescription, logger *logrus.Entry) (fs *FrameSource, err error) {
	fs = &FrameSource{
		offer:  offer,
		logger: logger.WithField("obj", "frame-source"),
	}

	fs.ctx, fs.cancel = context.WithCancel(ctx)

	fs.remoteStream, err = rtc.TransportFactory().NewRemoteStream(rtclib.RemoteStreamParams{
		Ctx:       fs.ctx,
		Logger:    fs.logger,
		PreferTCP: false,
	})

	if err != nil {
		fs.logger.WithError(err).Error("failed to create remote stream")
		return
	}

	fs.answer, err = fs.remoteStream.SetRemoteDescription(offer)
	if err != nil {
		fs.logger.WithError(err).Error("failed to set remote description")
		return
	}

	fs.metadata.ParseSdp(fs.answer.SDP)
	go fs.readRTP()

	return
}

func (fs *FrameSource) readRTP() {
	tracks, err := fs.remoteStream.GatheringTracks(true, true, 5*time.Second)
	if err != nil {
		fs.logger.WithError(err).Error("failed to gather tracks")
		return
	}

	for _, track := range tracks {
		go fs.readLoop(track)
	}
}

func (fs *FrameSource) readLoop(track *rtclib.TrackRemote) {
	for {
		select {
		case <-fs.ctx.Done():
			return
		default:
			frame, err := track.ReadRTP()
			if err != nil {
				fs.logger.WithError(err).Error("failed to read frame")
				return
			}

			fs.logger.WithField("frame", frame).Info("got frame")

		}
	}
}

func (fs *FrameSource) Answer() webrtc.SessionDescription {
	return fs.answer
}

func (fs *FrameSource) Metadata() *Metadata {
	return &fs.metadata
}
