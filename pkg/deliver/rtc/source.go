package rtc

import (
	"context"
	"fmt"
	"time"

	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pingostack/neon/pkg/rtclib"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"github.com/sirupsen/logrus"
)

type FrameSource struct {
	deliver.FrameSource
	ctx          context.Context
	cancel       context.CancelFunc
	remoteStream *rtclib.RemoteStream
	logger       *logrus.Entry
	//remoteSdp    webrtc.SessionDescription
	//localSdp webrtc.SessionDescription
	metadata deliver.Metadata
}

func NewFrameSource(ctx context.Context, streamFactory rtclib.StreamFactory, preferTCP bool, logger *logrus.Entry) (fs *FrameSource, err error) {
	if logger == nil {
		logger = logrus.WithField("obj", "frame-source")
	} else {
		logger = logger.WithField("obj", "frame-source")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			logger.WithError(err).Error("NewFrameSource panic")
		}

		if err != nil {
			if fs != nil && fs.remoteStream != nil {
				fs.remoteStream.Close()
			}
		}
	}()

	fs = &FrameSource{
		logger: logger,
	}

	fs.ctx, fs.cancel = context.WithCancel(ctx)

	fs.remoteStream, err = streamFactory.NewRemoteStream(rtclib.RemoteStreamParams{
		Ctx:       fs.ctx,
		Logger:    fs.logger,
		PreferTCP: preferTCP,
	})

	if err != nil {
		fs.logger.WithError(err).Error("failed to create remote stream")
		return nil, err
	}

	return fs, nil
}

func (fs *FrameSource) SetRemoteDescription(remoteSdp webrtc.SessionDescription) (localSdp webrtc.SessionDescription, err error) {
	//return fs.remoteStream.SetRemoteDescription(remoteSdp)

	localSdp, err = fs.remoteStream.SetRemoteDescription(remoteSdp)
	if err != nil {
		fs.logger.WithError(err).Error("failed to set remote description")
		return
	}

	fs.metadata = convMetadata(fs.remoteStream.PayloadUnion())
	fs.FrameSource = deliver.NewFrameSourceImpl(fs.ctx, fs.metadata.Audio.CodecType, fs.metadata.Video.CodecType, deliver.PacketTypeRtp)

	go fs.readRTP()

	return
}

func (fs *FrameSource) readRTP() {
	tracks, err := fs.remoteStream.GatheringTracks(true, true, 5*time.Second)
	if err != nil {
		fs.logger.WithError(err).Error("failed to gather tracks")
		return
	}

	keyFrameInterval := 2 * time.Second
	for _, track := range tracks {
		if track.IsVideo() {
			go func(vtrack *rtclib.TrackRemote) {
				keyframeTicker := time.NewTicker(keyFrameInterval)
				defer keyframeTicker.Stop()

				for range keyframeTicker.C {
					err := fs.remoteStream.PeerConnection.WriteRTCP([]rtcp.Packet{
						&rtcp.PictureLossIndication{

							MediaSSRC: uint32(vtrack.SSRC()),
						},
					})
					if err != nil {
						fs.logger.WithError(err).Error("failed to send pli")
						return
					}

					fs.logger.WithField("track", vtrack.SSRC()).Debug("send pli")
				}
			}(track)
		}
		go fs.readLoop(track)
	}
}

func (fs *FrameSource) readLoop(track *rtclib.TrackRemote) {
	for {
		select {
		case <-fs.ctx.Done():
			return
		default:
			rtpPacket, err := track.ReadRTP()
			if err != nil {
				fs.logger.WithError(err).Error("failed to read frame")
				return
			}

			//fs.logger.WithField("rtpPacket", rtpPacket).Debug("read rtp packet")

			var codec deliver.CodecType
			if track.IsAudio() {
				codec = deliver.ConvCodecType(fs.metadata.Audio.Codec)
			} else if track.IsVideo() {
				codec = deliver.ConvCodecType(fs.metadata.Video.Codec)
			}

			var additionalInfo deliver.FrameSpecificInfo
			if track.IsAudio() {
				additionalInfo = &deliver.AudioFrameSpecificInfo{
					SampleRate: fs.metadata.Audio.SampleRate,
				}
			} else if track.IsVideo() {
				additionalInfo = &deliver.VideoFrameSpecificInfo{}
			}

			frame := deliver.Frame{
				Codec:          codec,
				PacketType:     deliver.PacketTypeRtp,
				Length:         0,
				TimeStamp:      rtpPacket.Timestamp,
				AdditionalInfo: additionalInfo,
				RawPacket:      rtpPacket,
			}
			fs.DeliverFrame(frame, nil)
		}
	}
}

// func (fs *FrameSource) LocalSdp() webrtc.SessionDescription {
// 	return fs.localSdp
// }

func (fs *FrameSource) Metadata() *deliver.Metadata {
	return &fs.metadata
}
