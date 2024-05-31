package transport

//Part of the code is taken from https://github.com/livekit/livekit/blob/master/pkg/rtc/transport.go

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	lksdp "github.com/livekit/protocol/sdp"
	"github.com/pingostack/neon/pkg/eventemitter"
	"github.com/pingostack/neon/pkg/logger"
	"github.com/pingostack/neon/pkg/rtclib/config"
	"github.com/pingostack/neon/pkg/rtclib/rtcerror"
	"github.com/pingostack/neon/pkg/rtclib/sdpassistor"
	"github.com/pion/dtls/v2/pkg/crypto/elliptic"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

const (
	dtlsRetransmissionInterval = 100 * time.Millisecond
	iceDisconnectedTimeout     = 10 * time.Second // compatible for ice-lite with firefox client
	iceFailedTimeout           = 5 * time.Second  // time between disconnected and failed
	iceKeepaliveInterval       = 2 * time.Second  // pion's default
	defaultEventEmitterLength  = 10
)

var (
	signalLocalICECandidate    = eventemitter.GenEventID()
	signalRemoteICECandidate   = eventemitter.GenEventID()
	signalICEGatheringComplete = eventemitter.GenEventID()
	signalCloseTransport       = eventemitter.GenEventID()
)

type TransportOpt func(t *Transport)

type Transport struct {
	// input params
	webrtcConfig  *config.WebRTCConfig
	icc           *config.ICEConfig
	allowedCodecs []config.CodecConfig
	logger        logger.Logger
	eventemitter  eventemitter.EventEmitter
	ctx           context.Context
	cancel        context.CancelFunc

	*webrtc.PeerConnection
	preferTCP                  atomic.Bool
	lock                       sync.RWMutex
	iceConnectedAt             time.Time
	iceStartedAt               time.Time
	connectedAt                time.Time
	firstConnectedAt           time.Time
	signalingRTT               atomic.Uint32
	tcpICETimer                *time.Timer
	connectAfterICETimer       *time.Timer
	cacheLocalCandidates       bool
	cachedLocalCandidates      []*webrtc.ICECandidate
	allowedLocalCandidates     []string
	allowedRemoteCandidates    []string
	filteredLocalCandidates    []string
	filteredRemoteCandidates   []string
	currentOfferIceCredential  string
	onICECandidate             func(candidate webrtc.ICECandidateInit)
	onFailed                   func(isShort bool)
	onInitialConnected         func()
	onICEGathererStateComplete func()
	resetShortConnOnICERestart atomic.Bool
	pendingRemoteCandidates    []*webrtc.ICECandidateInit
	localSdpType               webrtc.SDPType
	localSdpSetted             bool
	remoteSdpSetted            bool
}

func NewTransport(opts ...TransportOpt) (*Transport, error) {
	t := &Transport{
		localSdpType: webrtc.SDPTypeOffer,
	}

	for _, opt := range opts {
		opt(t)
	}

	if err := t.validate(); err != nil {
		return nil, errors.Wrap(err, "invalid transport")
	}

	t.eventemitter.AddEvent(signalLocalICECandidate, t.handleLocalICECandidate)
	t.eventemitter.AddEvent(signalRemoteICECandidate, t.handleRemoteICECandidate)
	t.eventemitter.AddEvent(signalICEGatheringComplete, t.handleICEGatheringComplete)
	t.eventemitter.AddEvent(signalCloseTransport, t.handleCloseTransport)
	t.PeerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		t.logger.Debugf("ICE candidate: %s", candidate.String())

		t.eventemitter.EmitEvent(signalLocalICECandidate, candidate)
	})

	t.PeerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		t.logger.Debugf("ICE connection state changed: %s", state.String())
		switch state {
		case webrtc.ICEConnectionStateConnected:
			t.setICEConnectedAt(time.Now())
			pair, err := t.getSelectedPair()
			if err != nil {
				t.logger.Errorf("failed to get selected pair: %v", err)
			} else {
				t.logger.Infof("selected ice candidate pair: %s", pair.String())
			}
		case webrtc.ICEConnectionStateChecking:
			t.setICEStartedAt(time.Now())
		}
	})

	t.PeerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		t.logger.Debugf("ICE gathering state changed: %s", state.String())
		if state == webrtc.ICEGatheringStateComplete {
			t.eventemitter.EmitEvent(signalICEGatheringComplete, nil)
			if t.onICEGathererStateComplete != nil {
				t.onICEGathererStateComplete()
			}
		}
	})

	t.PeerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		t.logger.Debugf("Connection state changed: %s", state.String())
		switch state {
		case webrtc.PeerConnectionStateConnected:
			t.clearConnTimer()
			first := t.setConnectedAt(time.Now())
			if first {
				if t.onInitialConnected != nil {
					t.onInitialConnected()
				}
			}
		case webrtc.PeerConnectionStateFailed:
			t.logger.Warn("ICE connection failed",
				"lc", t.allowedLocalCandidates,
				"rc", t.allowedRemoteCandidates,
				"lc (filtered)", t.filteredLocalCandidates,
				"rc (filtered)", t.filteredRemoteCandidates)
			t.clearConnTimer()
			t.handleConnectionFailed(false)
		}
	})

	return t, nil
}

func WithConnConfig(icc *config.ICEConfig) TransportOpt {
	return func(t *Transport) {
		t.icc = icc
	}
}

func WithWebRTCConfig(webrtcConfig *config.WebRTCConfig) func(t *Transport) {
	return func(t *Transport) {
		t.webrtcConfig = webrtcConfig
	}
}

func WithAllowedCodecs(allowedCodecs []config.CodecConfig) func(t *Transport) {
	return func(t *Transport) {
		t.allowedCodecs = allowedCodecs
	}
}

func WithLogger(logger logger.Logger) func(t *Transport) {
	return func(t *Transport) {
		t.logger = logger
	}
}

func WithEventEmitter(eventemitter eventemitter.EventEmitter) func(t *Transport) {
	return func(t *Transport) {
		t.eventemitter = eventemitter
	}
}

func WithContext(ctx context.Context) func(t *Transport) {
	return func(t *Transport) {
		t.ctx, t.cancel = context.WithCancel(ctx)
	}
}

func (t *Transport) defaultICC() {
	icc := &config.ICEConfig{}
	if icc.ICEDisconnectedTimeout == 0 {
		icc.ICEDisconnectedTimeout = 10 * time.Second
	} else {
		icc.ICEDisconnectedTimeout *= time.Second
	}

	if icc.ICEFailedTimeout == 0 {
		icc.ICEFailedTimeout = 20 * time.Second
	} else {
		icc.ICEFailedTimeout *= time.Second
	}

	if icc.ICEKeepaliveInterval == 0 {
		icc.ICEKeepaliveInterval = 2 * time.Second
	} else {
		icc.ICEKeepaliveInterval *= time.Second
	}

	if icc.MinTcpICEConnectTimeout == 0 {
		icc.MinTcpICEConnectTimeout = 5 * time.Second
	} else {
		icc.MinTcpICEConnectTimeout *= time.Second
	}

	if icc.MaxTcpICEConnectTimeout == 0 {
		icc.MaxTcpICEConnectTimeout = 12 * time.Second
	} else {
		icc.MaxTcpICEConnectTimeout *= time.Second
	}

	if icc.ShortConnectionThreshold == 0 {
		icc.ShortConnectionThreshold = 90 * time.Second
	} else {
		icc.ShortConnectionThreshold *= time.Second
	}

	t.icc = icc
}

func (t *Transport) clearConnTimer() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.connectAfterICETimer != nil {
		t.connectAfterICETimer.Stop()
		t.connectAfterICETimer = nil
	}

	if t.tcpICETimer != nil {
		t.tcpICETimer.Stop()
		t.tcpICETimer = nil
	}
}

func (t *Transport) setConnectedAt(at time.Time) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.connectedAt = at

	if !t.firstConnectedAt.IsZero() {
		return false
	}

	t.firstConnectedAt = at

	return true
}

func (t *Transport) handleConnectionFailed(forceShortConn bool) {
	isShort := forceShortConn
	if !isShort {
		var duration time.Duration
		isShort, duration = t.isShortConnection(time.Now())
		if isShort {
			pair, err := t.getSelectedPair()
			if err != nil {
				t.logger.Errorf("Failed to get selected pair: %v, duration %d", err, duration)
			} else {
				t.logger.Infof("Short connection detected: %v, %v, duration %d", pair.Local, pair.Remote, duration)
			}
		}
	} else {
		t.logger.Infof("Force short connection detected")
	}

	if t.onFailed != nil {
		t.onFailed(isShort)
	}
}

func (t *Transport) getSelectedPair() (*webrtc.ICECandidatePair, error) {
	sctp := t.PeerConnection.SCTP()
	if sctp == nil {
		return nil, rtcerror.ErrEventNoSCTP
	}

	dtlsTransport := sctp.Transport()
	if dtlsTransport == nil {
		return nil, rtcerror.ErrNoDTLSTransport
	}

	iceTransport := dtlsTransport.ICETransport()
	if iceTransport == nil {
		return nil, rtcerror.ErrNoICETransport
	}

	return iceTransport.GetSelectedCandidatePair()
}

func (t *Transport) isShortConnection(at time.Time) (bool, time.Duration) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if t.iceConnectedAt.IsZero() {
		return false, 0
	}

	duration := at.Sub(t.iceConnectedAt)
	return duration < t.icc.ShortConnectionThreshold, duration
}

func (t *Transport) setICEStartedAt(tim time.Time) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if !t.iceStartedAt.IsZero() {
		return
	}

	t.iceStartedAt = tim

	if !t.preferTCP.Load() {
		return
	}

	signalingRTT := t.signalingRTT.Load()
	if signalingRTT < 1000 {
		tcpICETimeout := time.Duration(signalingRTT*8) * time.Millisecond
		if tcpICETimeout > t.icc.MaxTcpICEConnectTimeout {
			tcpICETimeout = t.icc.MaxTcpICEConnectTimeout
		} else if tcpICETimeout < t.icc.MinTcpICEConnectTimeout {
			tcpICETimeout = t.icc.MinTcpICEConnectTimeout
		}

		t.logger.Debugf("TCP ICE timeout: %s", tcpICETimeout.String())
		t.tcpICETimer = time.AfterFunc(tcpICETimeout, func() {
			if t.PeerConnection.ICEConnectionState() == webrtc.ICEConnectionStateChecking {
				t.logger.Debugf("TCP ICE timeout reached, close the connection")
				t.handleConnectionFailed(true)
			}
		})
	}
}

func (t *Transport) setICEConnectedAt(at time.Time) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if !t.iceConnectedAt.IsZero() {
		return
	}

	t.iceConnectedAt = at

	iceDuration := at.Sub(t.iceStartedAt)
	connTimeoutAfterICE := t.icc.MinTcpICEConnectTimeout
	if connTimeoutAfterICE < 3*iceDuration {
		connTimeoutAfterICE = 3 * iceDuration
	}

	if connTimeoutAfterICE > t.icc.MaxTcpICEConnectTimeout {
		connTimeoutAfterICE = t.icc.MaxTcpICEConnectTimeout
	}

	t.logger.Debugf("Setting TCP connection timeout: %s", connTimeoutAfterICE.String())
	t.connectAfterICETimer = time.AfterFunc(connTimeoutAfterICE, func() {
		t.logger.Debugf("TCP connection timeout reached, close the connection")
		state := t.PeerConnection.ConnectionState()
		if state != webrtc.PeerConnectionStateClosed &&
			state != webrtc.PeerConnectionStateFailed && !t.isEstablished() {
			t.logger.Infof("TCP connection timeout reached, timeout %d, ice duration %d", connTimeoutAfterICE, iceDuration)
			t.handleConnectionFailed(false)
		}
	})

	if t.tcpICETimer != nil {
		t.tcpICETimer.Stop()
		t.tcpICETimer = nil
	}
}

func (t *Transport) isEstablished() bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return !t.connectedAt.IsZero()
}

func (t *Transport) SetPreferTCP(preferTCP bool) {
	t.preferTCP.Store(preferTCP)
}

func (t *Transport) handleLocalICECandidate(data interface{}) {
	candidate := data.(*webrtc.ICECandidate)
	if candidate == nil {
		t.logger.Errorf("Local ICE candidate is nil")
		return
	}

	if t.preferTCP.Load() && candidate.Protocol != webrtc.ICEProtocolTCP {
		t.logger.Debugf("Local ICE candidate filtered out: %s", candidate.String())
		t.filteredLocalCandidates = append(t.filteredLocalCandidates, candidate.String())
		return
	}

	t.logger.Debugf("Local ICE candidate: %s", candidate.String())
	t.allowedLocalCandidates = append(t.allowedLocalCandidates, candidate.String())

	if t.cacheLocalCandidates {
		t.cachedLocalCandidates = append(t.cachedLocalCandidates, candidate)
		return
	}

	if t.onICECandidate != nil {
		t.onICECandidate(candidate.ToJSON())
	}
}

func (t *Transport) clearLocalDescriptionSent() {
	t.cacheLocalCandidates = true
	t.cachedLocalCandidates = nil
	t.allowedLocalCandidates = nil
	t.lock.Lock()
	t.allowedRemoteCandidates = nil
	t.lock.Unlock()
	t.filteredLocalCandidates = nil
	t.filteredRemoteCandidates = nil
}

func (t *Transport) handleRemoteICECandidate(data interface{}) {
	c := data.(*webrtc.ICECandidateInit)

	filtered := false
	if t.preferTCP.Load() && !strings.Contains(c.Candidate, "tcp") {
		t.filteredRemoteCandidates = append(t.filteredRemoteCandidates, c.Candidate)
		filtered = true
	}

	if filtered {
		return
	}

	t.lock.Lock()
	t.allowedRemoteCandidates = append(t.allowedRemoteCandidates, c.Candidate)
	t.lock.Unlock()

	if t.PeerConnection.RemoteDescription() == nil {
		t.pendingRemoteCandidates = append(t.pendingRemoteCandidates, c)
		return
	}

	if err := t.PeerConnection.AddICECandidate(*c); err != nil {
		// TODO: handle error
		return
	}

}

func (t *Transport) localDescriptionSent() error {
	if !t.cacheLocalCandidates {
		return nil
	}

	t.cacheLocalCandidates = false

	if t.onICECandidate == nil {
		for _, candidate := range t.cachedLocalCandidates {
			t.onICECandidate(candidate.ToJSON())
		}
	}

	t.cachedLocalCandidates = nil

	return nil
}

func (t *Transport) handleICEGatheringComplete(data interface{}) {
	t.logger.Debugf("ICE gathering complete")

}

func (t *Transport) UpdateICECredential(sdp *webrtc.SessionDescription) (bool, error) {
	parsed, err := sdp.Unmarshal()
	if err != nil {
		t.logger.Errorf("Failed to unmarshal SDP: %s", err)
		return false, err
	}

	user, pwd, err := lksdp.ExtractICECredential(parsed)
	if err != nil {
		t.logger.Errorf("Failed to extract ICE credential: %s", err)
		return false, err
	}

	credential := fmt.Sprintf("%s:%s", user, pwd)

	resetICE := (t.currentOfferIceCredential != "") && (t.currentOfferIceCredential != credential)

	t.logger.Infof("Update ICE credential: %s, instead of %s", credential, t.currentOfferIceCredential)

	t.currentOfferIceCredential = credential

	return resetICE, nil
}

func (t *Transport) resetShortConn() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.resetShortConnOnICERestart.CompareAndSwap(true, false) {
		t.iceStartedAt = time.Time{}
		t.iceConnectedAt = time.Time{}
		t.connectedAt = time.Time{}
		if t.tcpICETimer != nil {
			t.tcpICETimer.Stop()
			t.tcpICETimer = nil
		}

		if t.connectAfterICETimer != nil {
			t.connectAfterICETimer.Stop()
			t.connectAfterICETimer = nil
		}
	}
}

func (t *Transport) ResetShortConnOnICERestart() {
	t.resetShortConnOnICERestart.Store(true)
}

func (t *Transport) EnableRemoteCandidates() error {
	for _, c := range t.pendingRemoteCandidates {
		if e := t.PeerConnection.AddICECandidate(*c); e != nil {
			return errors.Wrap(rtcerror.ErrAddIceCandidate, e.Error())
		}
	}

	t.pendingRemoteCandidates = nil

	return nil
}

func (t *Transport) handleCloseTransport(data interface{}) {
	t.logger.Infof("Closing transport")
	if t.PeerConnection != nil {
		t.PeerConnection.Close()
	}
	t.cancel()
}

func (t *Transport) OnICECandidate(f func(candidate webrtc.ICECandidateInit)) {
	t.onICECandidate = f
}

func (t *Transport) OnICEGathererStateComplete(f func()) {
	t.onICEGathererStateComplete = f
}

func (t *Transport) OnInitialConnected(f func()) {
	t.onInitialConnected = f
}

func (t *Transport) OnFailed(f func(isShort bool)) {
	t.onFailed = f
}

func (t *Transport) validate() error {
	if t.ctx == nil {
		t.ctx, t.cancel = context.WithCancel(context.Background())
	}

	if t.logger == nil {
		t.logger = logger.DefaultLogger
	}

	if t.eventemitter == nil {
		t.eventemitter = eventemitter.NewEventEmitter(t.ctx, defaultEventEmitterLength, t.logger)
	}

	if t.icc == nil {
		t.defaultICC()
	}

	if t.webrtcConfig != nil {
		se := t.webrtcConfig.SettingEngine
		c := t.webrtcConfig.Configuration
		se.DisableMediaEngineCopy(true)
		// Change elliptic curve to improve connectivity
		// https://github.com/pion/dtls/pull/474
		se.SetDTLSEllipticCurves(elliptic.X25519, elliptic.P384, elliptic.P256)
		se.SetDTLSRetransmissionInterval(dtlsRetransmissionInterval)
		se.SetICETimeouts(iceDisconnectedTimeout, iceFailedTimeout, iceKeepaliveInterval)
		se.LoggerFactory = logger.NewPionLoggerFactory(t.logger)
		i := &interceptor.Registry{}

		me := CreateMediaEngine(t.allowedCodecs)
		if err := webrtc.RegisterDefaultInterceptors(me, i); err != nil {
			return errors.Wrap(err, "failed to register default interceptors")
		}

		api := webrtc.NewAPI(webrtc.WithMediaEngine(me),
			webrtc.WithInterceptorRegistry(i),
			webrtc.WithSettingEngine(se))

		PeerConnection, err := api.NewPeerConnection(c)
		if err != nil {
			return errors.Wrap(err, "failed to create peer connection")
		}

		t.PeerConnection = PeerConnection
	} else {
		PeerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			return errors.Wrap(err, "failed to create peer connection")
		}

		t.PeerConnection = PeerConnection
	}

	return nil
}

func (t *Transport) SetRemoteDescription(sd webrtc.SessionDescription) (err error) {
	if sd.Type == webrtc.SDPTypeOffer {
		t.localSdpType = webrtc.SDPTypeAnswer
	}

	sd, err = sdpassistor.FilterCandidates(sd, t.preferTCP.Load())
	if err != nil {
		err = errors.Wrap(err, "failed to filter sdp")
		return err
	}

	if err = t.PeerConnection.SetRemoteDescription(sd); err != nil {
		err = errors.Wrap(err, "failed to set remote description")
		return
	}

	t.remoteSdpSetted = true

	return
}

func (t *Transport) CreateOffer(options *webrtc.OfferOptions) (lsdp webrtc.SessionDescription, err error) {
	lsdp, err = t.PeerConnection.CreateOffer(options)
	if err != nil {
		err = errors.Wrap(err, "failed to create offer")
		return
	}

	if err = t.PeerConnection.SetLocalDescription(lsdp); err != nil {
		err = errors.Wrap(err, "failed to set local description")
		return
	}

	t.localSdpSetted = true

	return
}

func (t *Transport) CreateAnswer(options *webrtc.AnswerOptions) (lsdp webrtc.SessionDescription, err error) {
	lsdp, err = t.PeerConnection.CreateAnswer(options)
	if err != nil {
		err = errors.Wrap(err, "failed to create answer")
		return
	}

	if err = t.PeerConnection.SetLocalDescription(lsdp); err != nil {
		err = errors.Wrap(err, "failed to set local description")
		return
	}

	t.localSdpSetted = true

	return
}

func (t *Transport) CreateLocalDescription() (lsdp webrtc.SessionDescription, err error) {
	if t.localSdpType == webrtc.SDPTypeOffer {
		lsdp, err = t.PeerConnection.CreateOffer(nil)
		if err != nil {
			err = errors.Wrap(err, "failed to create answer")
			return
		}

		if err = t.PeerConnection.SetLocalDescription(lsdp); err != nil {
			err = errors.Wrap(err, "failed to set local description")
			return
		}
	} else {
		lsdp, err = t.PeerConnection.CreateAnswer(nil)
		if err != nil {
			err = errors.Wrap(err, "failed to create offer")
			return
		}

		if err = t.PeerConnection.SetLocalDescription(lsdp); err != nil {
			err = errors.Wrap(err, "failed to set local description")
			return
		}
	}

	t.localSdpSetted = true

	return
}

func (t *Transport) Context() context.Context {
	return t.ctx
}

func (t *Transport) Logger() logger.Logger {
	return t.logger
}

func (t *Transport) EventEmitter() eventemitter.EventEmitter {
	return t.eventemitter
}

func (t *Transport) Finalize() {
	t.eventemitter.EmitEvent(signalCloseTransport, nil)
}

func (t *Transport) LocalSdpType() webrtc.SDPType {
	return t.localSdpType
}

func (t *Transport) LocalSdpSetted() bool {
	return t.localSdpSetted
}

func (t *Transport) RemoteSdpSetted() bool {
	return t.remoteSdpSetted
}

func (t *Transport) GatheringCompleteLocalSdp(ctx context.Context) (lsdp webrtc.SessionDescription, err error) {
	if !t.localSdpSetted || !t.remoteSdpSetted {
		err = errors.New("local or remote sdp not set")
		return
	}

	// wait for gathering complete
	gatherComplete := webrtc.GatheringCompletePromise(t.PeerConnection)
	select {
	case <-ctx.Done():
		err = errors.Wrap(ctx.Err(), "context done")
		return
	case <-gatherComplete:
	}

	lsdp = *t.PeerConnection.LocalDescription()

	return
}
