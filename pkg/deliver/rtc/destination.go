package rtc

import (
	"context"
	"fmt"

	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pingostack/neon/pkg/rtclib"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/sirupsen/logrus"
)

type FrameDestination struct {
	deliver.FrameDestination
	ctx         context.Context
	cancel      context.CancelFunc
	localStream *rtclib.LocalStream
	logger      *logrus.Entry
	metadata    deliver.Metadata
	audioTrack  *webrtc.TrackLocalStaticRTP
	videoTrack  *webrtc.TrackLocalStaticRTP
}

func NewframeDestination(ctx context.Context, metadata deliver.Metadata, streamFactory rtclib.StreamFactory, preferTCP bool, logger *logrus.Entry) (fd *FrameDestination, err error) {
	if logger == nil {
		logger = logrus.WithField("obj", "frame-destination")
	} else {
		logger = logger.WithField("obj", "frame-destination")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			logger.WithError(err).Error("NewFrameDestination panic")
		}

		if err != nil {
			if fd != nil && fd.localStream != nil {
				fd.localStream.Close()
			}
		}
	}()

	fd = &FrameDestination{
		metadata: metadata,
		logger:   logger.WithField("obj", "frame-destination"),
	}

	fd.ctx, fd.cancel = context.WithCancel(ctx)

	fd.localStream, err = streamFactory.NewLocalStream(rtclib.LocalStreamParams{
		Ctx:       fd.ctx,
		Logger:    fd.logger,
		PreferTCP: preferTCP,
	})

	if err != nil {
		fd.logger.WithError(err).Error("failed to create local stream")
		return nil, err
	}

	fd.FrameDestination = deliver.NewFrameDestinationImpl(fd.ctx,
		fd.metadata.Audio.CodecType,
		fd.metadata.Video.CodecType,
		deliver.PacketTypeRtp)

	return fd, nil
}

func (fd *FrameDestination) processRTCP(rtpSender *webrtc.RTPSender) {
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
			fd.logger.WithError(rtcpErr).Error("failed to read rtcp packet")
			return
		}

		fd.logger.Debug("read rtcp packet")
	}
}

func (fd *FrameDestination) CreateOffer(options *webrtc.OfferOptions) (sd webrtc.SessionDescription, err error) {
	sd, err = fd.localStream.CreateOffer(options)
	if err != nil {
		return
	}

	if err = fd.localStream.SetLocalDescription(sd); err != nil {
		fd.logger.WithError(err).Error("failed to set local description")
		return
	}

	return sd, nil
}

func (fd *FrameDestination) SetRemoteDescription(remoteSdp webrtc.SessionDescription) (localSdp webrtc.SessionDescription, err error) {
	localSdp, err = fd.localStream.SetRemoteDescription(remoteSdp)
	if err != nil {
		fd.logger.WithError(err).Error("failed to set remote description")
		return webrtc.SessionDescription{}, err
	}

	fd.metadata = convMetadata(fd.localStream.PayloadUnion())

	return localSdp, nil
}

func (fd *FrameDestination) SetAudioSource(src deliver.FrameSource) error {
	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus, //fmt.Sprintf("audio/%s", src.SourceAudioCodec().String()),
			ClockRate: 48000,
			Channels:  2,
		}, "audio", "neon")
	if err != nil {
		return err
	}

	rtpSender, err := fd.localStream.AddTrack(track)
	if err != nil {
		return err
	}

	fd.audioTrack = track
	go fd.processRTCP(rtpSender)

	return fd.FrameDestination.SetAudioSource(src)
}

func (fd *FrameDestination) SetVideoSource(src deliver.FrameSource) error {
	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeVP8, //fmt.Sprintf("video/%s", src.SourceVideoCodec().String()),
			ClockRate: 90000,
		}, "video", "neon")
	if err != nil {
		return err
	}

	rtpSender, err := fd.localStream.AddTrack(track)
	if err != nil {
		return err
	}

	fd.videoTrack = track
	go fd.processRTCP(rtpSender)

	return fd.FrameDestination.SetVideoSource(src)
}

func (fd *FrameDestination) OnFrame(frame deliver.Frame, attr deliver.Attributes) {
	if frame.PacketType != deliver.PacketTypeRtp {
		fd.logger.WithField("packetType", frame.PacketType).Error("invalid packet type")
		return
	}

	var track *webrtc.TrackLocalStaticRTP
	if frame.Codec.IsAudio() {
		track = fd.audioTrack
	} else if frame.Codec.IsVideo() {
		track = fd.videoTrack
	}

	packet, ok := frame.RawPacket.(*rtp.Packet)
	if !ok {
		fd.logger.WithField("packet", frame.RawPacket).Error("invalid packet")
		return
	}

	//fd.logger.WithField("packet", packet).Debug("write rtp packet")

	err := track.WriteRTP(packet)
	if err != nil {
		fd.logger.WithError(err).Error("failed to write rtp packet")
	}
}
