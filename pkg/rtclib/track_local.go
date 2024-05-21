package rtclib

import (
	"context"
	"fmt"

	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pingostack/neon/pkg/logger"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

const (
	defaultWebrtcStreamID = "pingos"
)

type TrackLocl struct {
	track  *webrtc.TrackLocalStaticRTP
	ctx    context.Context
	cancel context.CancelFunc
	logger logger.Logger
	sender *webrtc.RTPSender
}

type addTrackFunc func(webrtc.TrackLocal) (*webrtc.RTPSender, error)

func NewTrackLocl(ctx context.Context, codec deliver.CodecType, clockRate uint32, addTrack addTrackFunc, logger logger.Logger) (*TrackLocl, error) {
	t := &TrackLocl{
		logger: logger,
	}

	t.ctx, t.cancel = context.WithCancel(ctx)

	switch codec {
	case deliver.CodecTypeAV1:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeAV1,
				ClockRate: 90000,
			},
			"av1",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case deliver.CodecTypeVP9:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeVP9,
				ClockRate: clockRate,
			},
			"vp9",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case deliver.CodecTypeVP8:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeVP8,
				ClockRate: clockRate,
			},
			"vp8",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case deliver.CodecTypeH264:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeH264,
				ClockRate: clockRate,
			},
			"h264",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case deliver.CodecTypeOpus:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeOpus,
				ClockRate: clockRate,
				Channels:  2,
			},
			"opus",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case deliver.CodecTypeG722_16000_2:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeG722,
				ClockRate: clockRate,
			},
			"g722",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case deliver.CodecTypePCMU:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypePCMU,
				ClockRate: clockRate,
			},
			"g711",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case deliver.CodecTypePCMA:

		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypePCMA,
				ClockRate: clockRate,
			},
			"g711",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported track type: %T", codec)
	}

	sender, err := addTrack(t.track)
	if err != nil {
		return nil, err
	}

	t.sender = sender

	return t, nil
}

func (t *TrackLocl) ReadRTCP(buf []byte) (n int, a interceptor.Attributes, err error) {
	return t.sender.Read(buf)
}

func (t *TrackLocl) WriteRTP(pkt *rtp.Packet) error {
	return t.track.WriteRTP(pkt)
}
