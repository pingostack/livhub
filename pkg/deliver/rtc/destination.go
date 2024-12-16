package rtc

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/pingostack/livhub/pkg/deliver"
	"github.com/pingostack/livhub/pkg/rtclib"
	"github.com/pingostack/livhub/pkg/rtclib/sdpassistor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type FrameDestination struct {
	deliver.FrameDestination
	*rtclib.LocalStream
	ctx                     context.Context
	cancel                  context.CancelFunc
	logger                  *logrus.Entry
	audioTrack              *rtclib.TrackLocl
	videoTrack              *rtclib.TrackLocl
	onceClose               sync.Once
	chSourceCompletePromise chan error
}

func NewFrameDestination(ctx context.Context, streamFactory rtclib.StreamFactory, preferTCP bool, logger *logrus.Entry) (fd *FrameDestination, err error) {
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
			if fd != nil && fd.LocalStream != nil {
				fd.LocalStream.Close()
			}
		}
	}()

	fd = &FrameDestination{
		chSourceCompletePromise: make(chan error, 1),
		logger:                  logger.WithField("obj", "frame-destination"),
	}

	fd.ctx, fd.cancel = context.WithCancel(ctx)

	fd.LocalStream, err = streamFactory.NewLocalStream(rtclib.LocalStreamParams{
		Ctx:       fd.ctx,
		Logger:    fd.logger,
		PreferTCP: preferTCP,
	})

	if err != nil {
		fd.logger.WithError(err).Error("failed to create local stream")
		return nil, err
	}

	return fd, nil
}

func (fd *FrameDestination) SetRemoteDescription(remoteSdp webrtc.SessionDescription) (err error) {
	err = fd.LocalStream.SetRemoteDescription(remoteSdp)
	if err != nil {
		fd.logger.WithError(err).Error("failed to set remote description")
		return errors.Wrap(err, "failed to set remote description")
	}

	if fd.LocalStream.LocalSdpType() == webrtc.SDPTypeAnswer {
		var payloadUnion *sdpassistor.PayloadUnion
		payloadUnion, err = sdpassistor.NewPayloadUnion(remoteSdp)
		if err != nil {
			fd.logger.WithError(err).Error("failed to create payload union")
			return errors.Wrap(err, "failed to create payload union")
		}

		fd.FrameDestination = deliver.NewFrameDestinationImpl(fd.ctx, convFormatSettings(payloadUnion))
	}

	return nil
}

func (fd *FrameDestination) CreateOffer(options *webrtc.OfferOptions) (webrtc.SessionDescription, error) {
	return fd.LocalStream.CreateOffer(options)
}

func (fd *FrameDestination) CreateAnswer(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
	return fd.LocalStream.CreateAnswer(options)
}

func (fd *FrameDestination) Start() error {
	if !fd.LocalStream.RemoteSdpSetted() || !fd.LocalStream.LocalSdpSetted() {
		return errors.New("remote sdp not setted or local sdp already setted")
	}

	return nil
}

func (fd *FrameDestination) AddAudioTrack(am *deliver.AudioMetadata) (err error) {
	if am == nil {
		return nil
	}

	fd.audioTrack, err = fd.LocalStream.AddTrack(am.CodecType, am.SampleRate, fd.logger)
	if err != nil {
		return err
	}

	go fd.loopReadRTCP(fd.audioTrack)

	return nil
}

func (fd *FrameDestination) AddVideoTrack(vm *deliver.VideoMetadata) (err error) {
	if vm == nil {
		return nil
	}

	fd.videoTrack, err = fd.LocalStream.AddTrack(vm.CodecType, vm.ClockRate, fd.logger)
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

	if err = fd.AddAudioTrack(src.Metadata().Audio); err != nil {
		return err
	}

	if err := fd.AddVideoTrack(src.Metadata().Video); err != nil {
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
		fd.LocalStream.Close()
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
