package rtc

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pingostack/neon/pkg/rtclib"
	"github.com/pingostack/neon/pkg/rtclib/sdpassistor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type FrameDestination struct {
	deliver.FrameDestination
	ctx                     context.Context
	cancel                  context.CancelFunc
	localStream             *rtclib.LocalStream
	logger                  *logrus.Entry
	audioTrack              *rtclib.TrackLocl
	videoTrack              *rtclib.TrackLocl
	onceClose               sync.Once
	chSourceCompletePromise chan error
}

func NewFrameDestination(ctx context.Context, streamFactory rtclib.StreamFactory, preferTCP bool, sd webrtc.SessionDescription, logger *logrus.Entry) (fd *FrameDestination, err error) {
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
		chSourceCompletePromise: make(chan error, 1),
		logger:                  logger.WithField("obj", "frame-destination"),
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

	payloadUnion, err := sdpassistor.NewPayloadUnion(sd)
	if err != nil {
		logger.WithError(err).Error("failed to create payload union")
		return nil, errors.Wrap(err, "failed to create payload union")
	}

	fd.FrameDestination = deliver.NewFrameDestinationImpl(fd.ctx, convFormatSettings(payloadUnion))

	return fd, nil
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

	return localSdp, nil
}

func (fd *FrameDestination) addAudioTrack(am *deliver.AudioMetadata) error {
	var err error
	fd.audioTrack, err = fd.localStream.AddTrack(am.CodecType, am.SampleRate, fd.logger)
	if err != nil {
		return err
	}

	go fd.loopReadRTCP(fd.audioTrack)

	return nil
}

func (fd *FrameDestination) addVideoTrack(vm *deliver.VideoMetadata) error {
	var err error
	fd.videoTrack, err = fd.localStream.AddTrack(vm.CodecType, vm.ClockRate, fd.logger)
	if err != nil {
		return err
	}

	go fd.loopReadRTCP(fd.videoTrack)

	return nil
}

func (fd *FrameDestination) OnSource(src deliver.FrameSource) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			fd.logger.WithError(err).Error("OnSource panic")
		}

		fd.chSourceCompletePromise <- err
	}()

	if err = fd.addAudioTrack(src.Metadata().Audio); err != nil {
		return err
	}

	if err := fd.addVideoTrack(src.Metadata().Video); err != nil {
		return err
	}

	err = fd.FrameDestination.OnSource(src)
	if err != nil {
		return err
	}

	return nil
}

func (fd *FrameDestination) OnFrame(frame deliver.Frame, attr deliver.Attributes) {
	defer func() {
		if r := recover(); r != nil {
			fd.logger.WithField("error", r).Error("OnFrame panic")
		}
	}()
	if frame.PacketType != deliver.PacketTypeRtp {
		fd.logger.WithField("packetType", frame.PacketType).Error("invalid packet type")
		return
	}

	var track *rtclib.TrackLocl
	if frame.Codec.IsAudio() {
		track = fd.audioTrack
		if track == nil {
			fd.logger.WithField("codec", frame.Codec).Error("audio track not found")
			return
		}
	} else if frame.Codec.IsVideo() {
		track = fd.videoTrack
		if track == nil {
			fd.logger.WithField("codec", frame.Codec).Error("video track not found")
			return
		}
	} else {
		fd.logger.WithField("codec", frame.Codec).Error("invalid codec")
		return
	}

	packet, ok := frame.RawPacket.(*rtp.Packet)
	if !ok {
		fd.logger.WithField("packet", frame.RawPacket).Error("invalid packet")
		return
	}

	err := track.WriteRTP(packet)
	if err != nil {
		fd.logger.WithError(err).Error("failed to write rtp packet")
	}
}

func (fd *FrameDestination) loopReadRTCP(track *rtclib.TrackLocl) {
	defer func() {
		if err := recover(); err != nil {
			fd.logger.WithField("error", err).Error("loopReadRTCP panic")
			fd.close()
		}
	}()

	buf := make([]byte, 1500)
	for {
		select {
		case <-fd.ctx.Done():
			return
		default:
			i, a, err := track.ReadRTCP(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					fd.logger.WithError(err).Info("read rtcp EOF")
					fd.close()
					return
				}

				fd.logger.WithError(err).Error("failed to read rtcp")
			} else {
				pkts, err := rtcp.Unmarshal(buf[:i])
				if err != nil {
					fd.logger.WithError(err).Error("failed to unmarshal rtcp")
					continue
				}

				for _, pkt := range pkts {
					switch p := pkt.(type) {
					case *rtcp.PictureLossIndication:
						fd.logger.WithField("ssrc", p.MediaSSRC).WithField("attri", a).Debug("received pli")
						fd.sendPLI()
					case *rtcp.FullIntraRequest:
						fd.logger.WithField("ssrc", p.MediaSSRC).WithField("attri", a).Debug("received fir")
						fd.sendFIR()
					default:
						//	fd.logger.WithField("pkt-type", reflect.TypeOf(pkt)).Debug("received rtcp")
					}
				}
			}

		}
	}
}

func (fd *FrameDestination) close() {
	fd.onceClose.Do(func() {
		fd.cancel()
		fd.localStream.Close()
		fd.FrameDestination.Close()
		fd.logger.Info("FrameDestination closed")
	})
}

func (fd *FrameDestination) Close() {
	fd.close()
}

func (fd *FrameDestination) sendPLI() {
	if fd.videoTrack == nil {
		return
	}

	fd.DeliverFeedback(deliver.FeedbackMsg{
		Type: deliver.FeedbackTypeVideo,
		Cmd:  deliver.FeedbackCmdPLI,
	})
}

func (fd *FrameDestination) sendFIR() {
	if fd.videoTrack == nil {
		return
	}

	fd.DeliverFeedback(deliver.FeedbackMsg{
		Type: deliver.FeedbackTypeVideo,
		Cmd:  deliver.FeedbackCmdFIR,
	})
}

func (fd *FrameDestination) SourceCompletePromise() <-chan error {
	return fd.chSourceCompletePromise
}
