package gortc

import (
	"context"
	"sync"
	"time"

	"github.com/bep/debounce"
	"github.com/let-light/neon/pkg/forwarder"
	"github.com/let-light/neon/pkg/utils"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type Subscriber struct {
	sync.Mutex
	OnOffer                    func(offer *webrtc.SessionDescription)
	OnIceCandidate             func(*webrtc.ICECandidateInit, int)
	OnICEConnectionStateChange func(webrtc.ICEConnectionState)
	onDownTrack                func(track forwarder.IDownTrack)
	id                         string
	logger                     *logrus.Entry
	pc                         *webrtc.PeerConnection
	candidates                 []webrtc.ICECandidateInit
	closeOnce                  sync.Once
	negotiate                  func()
	remoteAnswerPending        bool
	negotiationPending         bool
	closed                     utils.AtomicBool
	ctx                        context.Context
	cancel                     context.CancelFunc
}

func NewSubscriber(ctx context.Context, id string, c WebRTCTransportConfig, logger *logrus.Entry) (*Subscriber, error) {
	me, err := getSubscriberMediaEngine()
	if err != nil {
		logger.WithError(err).Error(err, "NewPeer error, getSubscriberMediaEngine")
		return nil, err
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithSettingEngine(c.Setting))
	pc, err := api.NewPeerConnection(c.Configuration)

	if err != nil {
		logger.WithError(err).Error("NewPeer error, NewPeerConnection")
		return nil, err
	}

	s := &Subscriber{
		id:     id,
		pc:     pc,
		logger: logger.WithField("role", "subscriber"),
		// tracks:          make(map[string][]*DownTrack),
		// channels:        make(map[string]*webrtc.DataChannel),
		// noAutoSubscribe: false,
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		logger.Info("ice connection status", "state", connectionState)
		switch connectionState {
		case webrtc.ICEConnectionStateFailed:
			fallthrough
		case webrtc.ICEConnectionStateClosed:
			s.closeOnce.Do(func() {
				logger.Info("webrtc ice closed")
				s.Close()
			})
		}
	})
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		s.Logger().Info("On subscriber ice candidate called for peer")
		if c == nil {
			return
		}

		if s.OnIceCandidate != nil && !s.closed.Get() {
			json := c.ToJSON()
			s.OnIceCandidate(&json, RoleSubscriber)
		}
	})

	s.OnNegotiateTracks(func() {
		s.Lock()
		defer s.Unlock()

		if s.remoteAnswerPending {
			s.negotiationPending = true
			return
		}

		s.Logger().Info("Negotiation needed")
		offer, err := s.CreateOffer()
		if err != nil {
			s.Logger().WithError(err).Error("CreateOffer error")
			return
		}

		s.remoteAnswerPending = true
		if s.OnOffer != nil && !s.closed.Get() {
			s.Logger().Info("OnOffer")
			s.OnOffer(&offer)
		}
	})

	//	go s.downTracksReports()

	subscriber := &Subscriber{
		id:     id,
		logger: logger,
	}

	return subscriber, nil
}

func (s *Subscriber) CreateOffer() (webrtc.SessionDescription, error) {
	offer, err := s.pc.CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	err = s.pc.SetLocalDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	return offer, nil
}

func (s *Subscriber) OnNegotiateTracks(f func()) {
	debounced := debounce.New(250 * time.Millisecond)
	s.negotiate = func() {
		debounced(f)
	}
}

func (s *Subscriber) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	if s.pc.RemoteDescription() != nil {
		return s.pc.AddICECandidate(candidate)
	}
	s.candidates = append(s.candidates, candidate)
	return nil
}

// SetRemoteDescription when receiving an answer from remote
func (s *Subscriber) SetRemoteDescription(sdp webrtc.SessionDescription) error {
	s.Lock()
	defer s.Unlock()

	s.Logger().Info("got answer")
	if err := s.pc.SetRemoteDescription(sdp); err != nil {
		s.Logger().WithError(err).Error("SetRemoteDescription error")
		return err
	}

	for _, c := range s.candidates {
		if err := s.pc.AddICECandidate(c); err != nil {
			s.Logger().WithError(err).Error("Add subscriber ice candidate to peer err")
		}
	}
	s.candidates = nil

	s.remoteAnswerPending = false

	if s.negotiationPending {
		s.negotiationPending = false
		s.negotiate()
	}

	return nil
}

func (s *Subscriber) Logger() *logrus.Entry {
	return s.logger
}

func (s *Subscriber) OnDownTrack(fn func(track forwarder.IDownTrack)) {
	s.onDownTrack = fn
}

func (s *Subscriber) Close() {
}

func (s *Subscriber) NegotiateTracks() {
	s.negotiate()
}
