package gortc

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/let-light/neon/pkg/buffer"
	"github.com/let-light/neon/pkg/forwarder"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type Publisher struct {
	OnIceCandidate                    func(*webrtc.ICECandidateInit, int)
	onICEConnectionStateChangeHandler atomic.Value
	onUpTrack                         func(track forwarder.IFrameSource, layer int8)
	peerId                            string
	logger                            *logrus.Entry
	pc                                *webrtc.PeerConnection
	cfg                               *WebRTCTransportConfig
	candidates                        []webrtc.ICECandidateInit
	closeOnce                         sync.Once
	factory                           *buffer.Factory
	ctx                               context.Context
	cancel                            context.CancelFunc
	rtcpCh                            chan []rtcp.Packet
	upTracks                          map[string]*UpTrack
}

func NewPublisher(ctx context.Context, id string, cfg WebRTCTransportConfig, logger *logrus.Entry) (*Publisher, error) {
	me, err := getPublisherMediaEngine()
	if err != nil {
		logger.WithError(err).Error("NewPublisher error, getPublisherMediaEngine")
		return nil, err
	}

	factory := buffer.NewBufferFactory(cfg.TrackingPackets, logrus.WithField("package", "buffer"))
	logger.Infof("NewPublisher, cfg %+v", cfg.Configuration)
	cfg.Setting.BufferFactory = factory.GetOrNew

	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithSettingEngine(cfg.Setting))
	pc, err := api.NewPeerConnection(cfg.Configuration)
	if err != nil {
		logger.WithError(err).Error("NewPublisher error, NewPeerConnection")
		return nil, err
	}

	publisher := &Publisher{
		peerId:  id,
		logger:  logger.WithField("role", "publisher"),
		pc:      pc,
		cfg:     &cfg,
		factory: factory,
	}

	publisher.ctx, publisher.cancel = context.WithCancel(ctx)

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		logger.Info("on publisher ice candidate called for peer")
		if c == nil {
			return
		}

		if publisher.OnIceCandidate != nil {
			iceInit := c.ToJSON()
			publisher.OnIceCandidate(&iceInit, RolePublisher)
		}
	})

	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		publisher.Logger().WithField("connectionState", connectionState).Info("ice connection status")
		switch connectionState {
		case webrtc.ICEConnectionStateFailed:
			fallthrough
		case webrtc.ICEConnectionStateClosed:
			publisher.Logger().Info("webrtc ice closed")
			publisher.Close()
		}

		if handler, ok := publisher.onICEConnectionStateChangeHandler.Load().(func(webrtc.ICEConnectionState)); ok && handler != nil {
			handler(connectionState)
		}
	})

	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		publisher.Logger().WithFields(logrus.Fields{
			"publisher_id": publisher.peerId,
			"track_id":     track.ID(),
			"mediaSSRC":    track.SSRC(),
			"rid":          track.RID(),
			"stream_id":    track.StreamID(),
		}).Info("Peer got remote track id")

		buff := publisher.factory.GetBuffer(uint32(track.SSRC()))
		if buff == nil {
			publisher.Logger().WithField("track_id", track.ID()).Error("buffer is nil")
			return
		}

		up := NewUpTrack(buff, track, receiver, uint64(publisher.cfg.MaxBitRate), publisher.logger)
		up.SetRTCPCh(publisher.rtcpCh)
		if publisher.onUpTrack != nil {
			publisher.onUpTrack(up, up.Layer())
		}
	})

	return publisher, nil
}

func (p *Publisher) GetUpTrack(trackId string) *UpTrack {
	return p.upTracks[trackId]
}

func (p *Publisher) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	p.Logger().Info("add publisher ice candidate called for peer")
	if p.pc.RemoteDescription() != nil {
		p.Logger().Info("add publisher ice candidate called for peer, remote description is not nil")
		return p.pc.AddICECandidate(candidate)
	}
	p.candidates = append(p.candidates, candidate)
	return nil
}

func (p *Publisher) Answer(sdp webrtc.SessionDescription) (*webrtc.SessionDescription, error) {

	p.Logger().Infof("got offer sdp: %s", sdp.SDP)

	if p.pc.SignalingState() != webrtc.SignalingStateStable {
		return nil, fmt.Errorf("signalstate is not stable")
	}

	if err := p.pc.SetRemoteDescription(sdp); err != nil {
		return nil, err
	}

	for _, c := range p.candidates {
		if err := p.pc.AddICECandidate(c); err != nil {
			p.Logger().WithError(err).Error("Add publisher ice candidate to peer err")
		}
	}

	p.Logger().Infof("publisher candidates: %v", p.candidates)
	p.candidates = nil

	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}

	if err := p.pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

func (p *Publisher) Logger() *logrus.Entry {
	return p.logger
}

func (p *Publisher) sendRTCPLoop() {
	for {
		select {
		case pkts := <-p.rtcpCh:
			p.pc.WriteRTCP(pkts)
		case <-p.ctx.Done():
			return
		}
	}
}

// IPublisher impl begin
func (p *Publisher) OnUpTrack(fn func(track forwarder.IFrameSource, layer int8)) {
	p.onUpTrack = fn
}

func (p *Publisher) Close() {
	p.closeOnce.Do(func() {
		p.cancel()
		p.Logger().Info("publisher close")

		if err := p.pc.Close(); err != nil {
			p.Logger().WithError(err).Error("webrtc transport close err")
		}
	})
}

// IPublisher impl end
