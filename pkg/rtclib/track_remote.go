package rtclib

import (
	"context"

	"github.com/pingostack/neon/pkg/logger"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type TrackRemote struct {
	ctx      context.Context
	track    *webrtc.TrackRemote
	receiver *webrtc.RTPReceiver
	logger   logger.Logger
}

func NewTrackRemote(ctx context.Context,
	track *webrtc.TrackRemote,
	receiver *webrtc.RTPReceiver,
	writeRTCP func([]rtcp.Packet) error,
	logger logger.Logger) *TrackRemote {
	t := &TrackRemote{
		ctx:      ctx,
		track:    track,
		receiver: receiver,
		logger:   logger,
	}

	return t
}

func (t *TrackRemote) ReadRTCP(buf []byte) (n int, a interceptor.Attributes, err error) {
	return t.receiver.Read(buf)
}

func (t *TrackRemote) ReadRTP() (*rtp.Packet, error) {
	packet, _, err := t.track.ReadRTP()
	return packet, err
}

func (t *TrackRemote) IsAudio() bool {
	return t.track.Kind() == webrtc.RTPCodecTypeAudio
}

func (t *TrackRemote) IsVideo() bool {
	return t.track.Kind() == webrtc.RTPCodecTypeVideo
}

func (t *TrackRemote) SSRC() webrtc.SSRC {
	return t.track.SSRC()
}

func (t *TrackRemote) Kind() webrtc.RTPCodecType {
	return t.track.Kind()
}
