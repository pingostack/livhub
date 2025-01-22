package gortc

import (
	"fmt"
	"io"
	"sync"

	"github.com/im-pingo/mediatransportutil/pkg/rtcconfig"
	"github.com/pingostack/livhub/pkg/logger"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

// PeerType represents the type of peer (publisher or subscriber)
type PeerType int

const (
	PeerTypePublisher PeerType = iota
	PeerTypeSubscriber
)

// PeerConfig represents the configuration for a WebRTC peer
type PeerConfig struct {
	RTCConfig   *rtcconfig.WebRTCConfig
	TrickleICE  bool
	IsPublisher bool
}

// TrackInfo contains information about a media track
type TrackInfo struct {
	ID          string
	Kind        webrtc.RTPCodecType
	CodecParams webrtc.RTPCodecCapability
}

// Peer represents a WebRTC peer connection with additional functionality
type Peer struct {
	id             string
	peerType       PeerType
	conn           *webrtc.PeerConnection
	tracks         sync.Map // string -> *webrtc.TrackLocalStaticRTP
	remoteTracks   sync.Map // string -> *webrtc.TrackRemote
	onTrackChan    chan TrackInfo
	onCloseChan    chan struct{}
	onICECandidate chan *webrtc.ICECandidate
	log            logger.Logger
	mu             sync.Mutex
	trickleICE     bool
}

// NewPeer creates a new WebRTC peer
func NewPeer(id string, peerType PeerType, config PeerConfig) (*Peer, error) {
	var api *webrtc.API

	// Create API with WebRTC config
	if config.RTCConfig != nil {
		api = webrtc.NewAPI(webrtc.WithSettingEngine(config.RTCConfig.SettingEngine))
	} else {
		api = webrtc.NewAPI()
	}

	// Create WebRTC configuration
	webrtcConfig := webrtc.Configuration{
		ICEServers:         config.RTCConfig.Configuration.ICEServers,
		ICETransportPolicy: config.RTCConfig.Configuration.ICETransportPolicy,
	}

	// Create a new PeerConnection using the API
	peerConnection, err := api.NewPeerConnection(webrtcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	peer := &Peer{
		id:             id,
		peerType:       peerType,
		conn:           peerConnection,
		onTrackChan:    make(chan TrackInfo, 10),
		onCloseChan:    make(chan struct{}),
		onICECandidate: make(chan *webrtc.ICECandidate, 10),
		log:            logger.WithField("module", "webrtc").WithField("peer_id", id),
		trickleICE:     config.TrickleICE,
	}

	// Set up callbacks
	peer.setupCallbacks()

	return peer, nil
}

// SetRemoteDescription sets the remote description for the peer
func (p *Peer) SetRemoteDescription(sdp string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}

	if err := p.conn.SetRemoteDescription(offer); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	// Create answer
	answer, err := p.conn.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("failed to create answer: %w", err)
	}

	// Set local description
	if err = p.conn.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	return nil
}

// GetLocalDescription returns the local description (answer)
func (p *Peer) GetLocalDescription() string {
	return p.conn.LocalDescription().SDP
}

// AddTrack adds a new track to the peer connection
func (p *Peer) AddTrack(trackInfo TrackInfo) (*webrtc.TrackLocalStaticRTP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	track, err := webrtc.NewTrackLocalStaticRTP(trackInfo.CodecParams, trackInfo.ID, trackInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create track: %w", err)
	}

	sender, err := p.conn.AddTrack(track)
	if err != nil {
		return nil, fmt.Errorf("failed to add track: %w", err)
	}

	// Start RTP sender read routine to handle RTCP packets
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, err := sender.Read(rtcpBuf); err != nil {
				if err == io.EOF {
					return
				}
				p.log.WithError(err).Error("Failed to read RTCP packet")
			}
		}
	}()

	p.tracks.Store(trackInfo.ID, track)
	return track, nil
}

// OnTrack returns a channel that receives track information when new tracks are added
func (p *Peer) OnTrack() <-chan TrackInfo {
	return p.onTrackChan
}

// OnClose returns a channel that is closed when the peer connection is closed
func (p *Peer) OnClose() <-chan struct{} {
	return p.onCloseChan
}

// Close closes the peer connection and cleans up resources
func (p *Peer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.onCloseChan)

	if err := p.conn.Close(); err != nil {
		return fmt.Errorf("failed to close peer connection: %w", err)
	}

	return nil
}

// WriteRTP writes RTP packets to a specific track
func (p *Peer) WriteRTP(trackID string, packet *rtp.Packet) error {
	trackValue, ok := p.tracks.Load(trackID)
	if !ok {
		return fmt.Errorf("track not found: %s", trackID)
	}

	track := trackValue.(*webrtc.TrackLocalStaticRTP)
	if err := track.WriteRTP(packet); err != nil {
		return fmt.Errorf("failed to write RTP: %w", err)
	}

	return nil
}

// ReadRTP reads RTP packets from a specific track
func (p *Peer) ReadRTP(trackID string) (*webrtc.TrackRemote, error) {
	trackValue, ok := p.remoteTracks.Load(trackID)
	if !ok {
		return nil, fmt.Errorf("remote track not found: %s", trackID)
	}

	return trackValue.(*webrtc.TrackRemote), nil
}

// CreateOffer creates an offer and sets it as local description
func (p *Peer) CreateOffer() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	offer, err := p.conn.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create offer: %w", err)
	}

	if err = p.conn.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("failed to set local description: %w", err)
	}

	return offer.SDP, nil
}

// SetAnswer sets a remote answer
func (p *Peer) SetAnswer(sdp string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}

	if err := p.conn.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("failed to set remote answer: %w", err)
	}

	return nil
}

// RestartICE restarts the ICE connection by creating a new offer with ICE restart
func (p *Peer) RestartICE() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	offer, err := p.conn.CreateOffer(&webrtc.OfferOptions{
		ICERestart: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create offer with ICE restart: %w", err)
	}

	if err = p.conn.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	return nil
}

// OnICECandidate returns a channel that receives ICE candidates
func (p *Peer) OnICECandidate() <-chan *webrtc.ICECandidate {
	return p.onICECandidate
}

// AddICECandidate adds a remote ICE candidate
func (p *Peer) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	if err := p.conn.AddICECandidate(candidate); err != nil {
		return fmt.Errorf("failed to add ICE candidate: %w", err)
	}
	return nil
}

// setupCallbacks sets up the WebRTC callbacks
func (p *Peer) setupCallbacks() {
	// Handle ICE connection state changes
	p.conn.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		p.log.WithField("state", state.String()).Info("ICE connection state changed")
		if state == webrtc.ICEConnectionStateFailed {
			p.Close()
		}
	})

	// Handle ICE candidates if trickle ICE is enabled
	if p.trickleICE {
		p.conn.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			if candidate != nil {
				p.onICECandidate <- candidate
			}
		})
	}

	// Handle incoming tracks
	p.conn.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		p.log.WithField("track_id", track.ID()).Info("New track received")

		p.remoteTracks.Store(track.ID(), track)

		p.onTrackChan <- TrackInfo{
			ID:          track.ID(),
			Kind:        track.Kind(),
			CodecParams: track.Codec().RTPCodecCapability,
		}
	})
}
