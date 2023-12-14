package rtclib

import (
	"context"

	"github.com/pingostack/neon/pkg/logger"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
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

	go t.loopReadRTCP()

	return t
}

func (t *TrackRemote) loopReadRTCP() {
	buf := make([]byte, 1500)
	for {
		_, _, err := t.receiver.Read(buf)
		if err != nil {
			t.logger.Warnf("receiver read rtcp error %v", err)
			return
		}
	}
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
