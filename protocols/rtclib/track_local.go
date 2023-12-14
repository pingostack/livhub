package rtclib

import (
	"fmt"

	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

const (
	defaultWebrtcStreamID = "pingos"
)

type TrackLocl struct {
	track *webrtc.TrackLocalStaticRTP
}

type addTrackFunc func(webrtc.TrackLocal) (*webrtc.RTPSender, error)

func NewTrackLocl(forma format.Format, addTrack addTrackFunc) (*TrackLocl, error) {
	t := &TrackLocl{}

	switch forma := forma.(type) {
	case *format.AV1:
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

	case *format.VP9:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeVP9,
				ClockRate: uint32(forma.ClockRate()),
			},
			"vp9",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case *format.VP8:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeVP8,
				ClockRate: uint32(forma.ClockRate()),
			},
			"vp8",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case *format.H264:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeH264,
				ClockRate: uint32(forma.ClockRate()),
			},
			"h264",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case *format.Opus:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeOpus,
				ClockRate: uint32(forma.ClockRate()),
				Channels:  2,
			},
			"opus",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case *format.G722:
		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  webrtc.MimeTypeG722,
				ClockRate: uint32(forma.ClockRate()),
			},
			"g722",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	case *format.G711:
		var mtyp string
		if forma.MULaw {
			mtyp = webrtc.MimeTypePCMU
		} else {
			mtyp = webrtc.MimeTypePCMA
		}

		var err error
		t.track, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{
				MimeType:  mtyp,
				ClockRate: uint32(forma.ClockRate()),
			},
			"g711",
			defaultWebrtcStreamID,
		)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported track type: %T", forma)
	}

	sender, err := addTrack(t.track)
	if err != nil {
		return nil, err
	}

	go t.loopReadRTCP(sender)

	return t, nil
}

func (t *TrackLocl) loopReadRTCP(sender *webrtc.RTPSender) {
	buf := make([]byte, 1500)
	for {
		_, _, err := sender.Read(buf)
		if err != nil {
			return
		}
	}
}

func (t *TrackLocl) WriteRTP(pkt *rtp.Packet) error {
	return t.track.WriteRTP(pkt)
}
