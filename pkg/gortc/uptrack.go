package gortc

import (
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/let-light/neon/pkg/buffer"
	"github.com/let-light/neon/pkg/forwarder"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type UpTrack struct {
	forwarder.FrameSource
	buff         *buffer.Buffer
	track        *webrtc.TrackRemote
	receiver     *webrtc.RTPReceiver
	maxBitRate   uint64
	logger       *logrus.Entry
	rtcpCh       chan []rtcp.Packet
	sendRTCPOnce sync.Once
	lastPli      int64
	nackWorker   *workerpool.WorkerPool
	format       forwarder.FrameFormat
	layer        int8
}

func NewUpTrack(buff *buffer.Buffer,
	remoteTrack *webrtc.TrackRemote,
	receiver *webrtc.RTPReceiver,
	maxBitRate uint64,
	logger *logrus.Entry) *UpTrack {
	track := &UpTrack{
		buff:       buff,
		track:      remoteTrack,
		receiver:   receiver,
		maxBitRate: maxBitRate,
		logger:     logger,
		nackWorker: workerpool.New(1),
		format:     convFormart(receiver.GetParameters().Codecs[0].MimeType),
		layer:      convLayer(remoteTrack.RID()),
	}

	buff.OnTransportWideCC(func(sn uint16, timeNS int64, marker bool) {
		track.logger.Infof("sn %d", sn)
	})

	buff.Bind(receiver.GetParameters(), buffer.Options{
		MaxBitRate: maxBitRate,
	})

	go track.readExtendedLoop()
	return track
}

func (up *UpTrack) convFrame(rtpPacket *buffer.ExtPacket) *forwarder.Frame {
	frame := &forwarder.Frame{
		PT: forwarder.PacketTypeRtp,
		// TODO: 这里很重要，这里直接讲Codecs数组里的第一个元素作为FrameFormat，但是这里有个问题，如果Codecs数组里有多个元素，那么这里可能就会出问题
		Format:    up.format,
		Layer:     up.layer,
		Timestamp: uint64(rtpPacket.Packet.Header.Timestamp),
		// TODO: VFSI AFSI 这两个参数还没有处理
		Payload: rtpPacket,
	}

	return frame
}

func (up *UpTrack) readExtendedLoop() {
	for {
		rtpPacket, err := up.buff.ReadExtended()
		if err == io.EOF {
			return
		}

		up.logger.Debugf("read packet %+v", rtpPacket)

		frame := up.convFrame(rtpPacket)

		up.DeliverFrame(frame)
	}
}

func (up *UpTrack) SetRTCPCh(ch chan []rtcp.Packet) {
	up.rtcpCh = ch

	up.buff.OnFeedback(func(fb []rtcp.Packet) {
		up.rtcpCh <- fb
	})
}

func (up *UpTrack) SendRTCP(p []rtcp.Packet) {
	if _, ok := p[0].(*rtcp.PictureLossIndication); ok {
		if time.Now().UnixNano()-atomic.LoadInt64(&up.lastPli) < 500e6 {
			return
		}
		atomic.StoreInt64(&up.lastPli, time.Now().UnixNano())
	}

	up.rtcpCh <- p
}

func (up *UpTrack) GetFrameKind() forwarder.FrameKind {
	return forwarder.FrameKindVideo
}

func (up *UpTrack) GetPacketType() forwarder.PacketType {
	return forwarder.PacketTypeRtp
}

func (up *UpTrack) GetCodecType() forwarder.FrameFormat {
	return forwarder.FrameFormatH264
}

func (up *UpTrack) Close() {

}

func (up *UpTrack) Layer() int8 {
	return 0
}

func (up *UpTrack) WriteFeedback(fb *forwarder.FeedbackMsg) {
	switch fb.Type {
	case forwarder.FeedbackTypePLI:
		pli := []rtcp.Packet{
			&rtcp.PictureLossIndication{SenderSSRC: rand.Uint32(), MediaSSRC: uint32(up.track.SSRC())},
		}
		up.SendRTCP(pli)
	case forwarder.FeedbackTypeNack:
		up.nackWorker.Submit(func() {
			msg := fb.Data.(RetransmitPacketMsg)
			src := rtpPacketPool.Get().(*[]byte)
			for _, meta := range msg.Packets {
				pktBuff := *src
				buff := up.buff
				if buff == nil {
					break
				}
				i, err := buff.GetPacket(pktBuff, meta.sourceSeqNo)
				if err != nil {
					if err == io.EOF {
						break
					}
					continue
				}
				var pkt rtp.Packet
				if err = pkt.Unmarshal(pktBuff[:i]); err != nil {
					continue
				}
				pkt.Header.SequenceNumber = meta.targetSeqNo
				pkt.Header.Timestamp = meta.timestamp
				pkt.Header.SSRC = meta.ssrc
				pkt.Header.PayloadType = meta.payloadType
				if meta.temporalSupported {
					switch up.format {
					case forwarder.FrameFormatVP8:
						var vp8 buffer.VP8
						if err = vp8.Unmarshal(pkt.Payload); err != nil {
							continue
						}
						tlzoID, picID := meta.getVP8PayloadMeta()
						modifyVP8TemporalPayload(pkt.Payload, vp8.PicIDIdx, vp8.TlzIdx, picID, tlzoID, vp8.MBit)
					}
				}

				msg.Dest.WriteFrame(up.convFrame(&pkt))
				//if _, err = track.writeStream.WriteRTP(&pkt.Header, pkt.Payload); err != nil {
			}
			rtpPacketPool.Put(src)

		})
	}
}
