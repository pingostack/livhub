package rtc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
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
	metadata         deliver.Metadata
	keyFrameInterval time.Duration
	videoTrack       *rtclib.TrackRemote
	audioTrack       *rtclib.TrackRemote
	onceClose        sync.Once
}

func NewFrameSource(ctx context.Context, streamFactory rtclib.StreamFactory, preferTCP bool, keyFrameInterval time.Duration, logger *logrus.Entry) (fs *FrameSource, err error) {
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
		keyFrameInterval: keyFrameInterval,
		logger:           logger,
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
	fs.FrameSource = deliver.NewFrameSourceImpl(fs.ctx, fs.metadata)

	go fs.readRTP()

	return
}

func (fs *FrameSource) gatheringTracks() error {
	tracks, err := fs.remoteStream.GatheringTracks(true, true, 20*time.Second)
	if err != nil {
		return err
	}

	for _, track := range tracks {
		if track.IsAudio() {
			fs.audioTrack = track
		} else if track.IsVideo() {
			fs.videoTrack = track
		}
	}

	return nil
}

func (fs *FrameSource) readRTP() {

	err := fs.gatheringTracks()
	if err != nil {
		fs.logger.WithError(err).Error("failed to gather tracks")
		return
	}

	if fs.audioTrack != nil {
		go fs.loopReadRTP(fs.audioTrack)
		go fs.loopReadRTCP(fs.audioTrack)
	}

	if fs.videoTrack != nil {
		go fs.loopReadRTP(fs.videoTrack)
		go fs.loopReadRTCP(fs.videoTrack)

		if fs.keyFrameInterval > 0 {
			go fs.cycleKeyframe()
		}
	}
}

func (fs *FrameSource) cycleKeyframe() {
	for {
		select {
		case <-fs.ctx.Done():
			return
		case <-time.After(fs.keyFrameInterval):
			fs.sendPLI()
		}
	}
}

func (fs *FrameSource) sendPLI() {
	if fs.videoTrack == nil {
		return
	}

	err := fs.remoteStream.PeerConnection.WriteRTCP([]rtcp.Packet{
		&rtcp.PictureLossIndication{
			MediaSSRC: uint32(fs.videoTrack.SSRC()),
		},
	})
	if err != nil {
		fs.logger.WithError(err).Error("failed to send pli")
		return
	}

	fs.logger.WithField("track", fs.videoTrack.SSRC()).Debug("send pli")
}

func (fs *FrameSource) loopReadRTCP(track *rtclib.TrackRemote) {
	defer func() {
		if r := recover(); r != nil {
			fs.logger.WithField("error", r).Error("loopReadRTCP panic")
			fs.close()
		}
	}()

	buf := make([]byte, 1500)
	for {
		select {
		case <-fs.ctx.Done():
			return
		default:
			_, _, err := track.ReadRTCP(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					fs.logger.WithError(err).Info("read rtcp EOF")
					fs.close()
					return
				}

				fs.logger.WithError(err).Error("failed to read rtcp")
			}

		}
	}
}

func (fs *FrameSource) loopReadRTP(track *rtclib.TrackRemote) {
	defer func() {
		if r := recover(); r != nil {
			fs.logger.WithField("error", r).Error("loopReadRTP panic")
			fs.close()
		}
	}()

	for {
		select {
		case <-fs.ctx.Done():
			return
		default:
			rtpPacket, err := track.ReadRTP()
			if err != nil {
				if errors.Is(err, io.EOF) {
					fs.logger.WithField("track", track.SSRC()).Info("read rtp EOF")
					fs.close()
					return
				}
				fs.logger.WithError(err).Error("failed to read frame")
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

func (fs *FrameSource) OnFeedback(feedback deliver.FeedbackMsg) {
	if feedback.Type != deliver.FeedbackTypeVideo {
		return
	}

	if fs.videoTrack == nil {
		return
	}

	switch feedback.Cmd {
	case deliver.FeedbackCmdPLI:
		fs.sendPLI()
	}

}

func (fs *FrameSource) close() {
	fs.onceClose.Do(func() {
		fs.cancel()
		fs.remoteStream.Close()
		fs.FrameSource.Close()
		fs.logger.Debug("FrameSource closed")
	})
}

func (fs *FrameSource) Close() {
	fs.close()
}
