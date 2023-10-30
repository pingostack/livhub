package forwarder

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/let-light/neon/pkg/utils"
	"github.com/sirupsen/logrus"
)

type Multicast struct {
	upTracks       [3]IFrameSource
	available      [3]utils.AtomicBool
	downTracks     [3]atomic.Value // []IDownTrack
	pending        [3]utils.AtomicBool
	pendingTracks  [3][]IDownTrack
	closed         utils.AtomicBool
	mutex          sync.Mutex
	peerID         string
	pt             PacketType
	format          FrameFormat
	isSimulcast    bool
	closeOnce      sync.Once
	logger         *logrus.Entry
	filter         IFilter
	onCloseHandler func()
}

func NewMulticast(peerID string, pt PacketType, format FrameFormat, isSimulcast bool) *Multicast {
	return &Multicast{
		peerID:      peerID,
		pt:          pt,
		format:       format,
		isSimulcast: isSimulcast,
		logger:      logrus.WithFields(logrus.Fields{"package": "forwarder", "role": "multicast", "peer_id": peerID}),
	}
}

func (m *Multicast) AddFilter(filter IFilter) {
	m.filter = filter
}

func (m *Multicast) AddUpTrack(src IFrameSource, layer int, bestQualityFirst bool) {
	if m.closed.Get() {
		return
	}

	m.mutex.Lock()
	m.upTracks[layer] = src
	m.available[layer].Set(true)
	m.downTracks[layer].Store(make([]IDownTrack, 0, 10))
	m.pendingTracks[layer] = make([]IDownTrack, 0, 10)
	m.mutex.Unlock()

	subBestQuality := func(targetLayer int) {
		for l := 0; l < targetLayer; l++ {
			dts := m.downTracks[l].Load()
			if dts == nil {
				continue
			}
			for _, dt := range dts.([]IDownTrack) {
				_ = dt.SwitchSpatialLayer(targetLayer, false)
			}
		}
	}

	subLowestQuality := func(targetLayer int) {
		for l := 2; l != targetLayer; l-- {
			dts := m.downTracks[l].Load()
			if dts == nil {
				continue
			}
			for _, dt := range dts.([]IDownTrack) {
				_ = dt.SwitchSpatialLayer(targetLayer, false)
			}
		}
	}

	if m.isSimulcast {
		if bestQualityFirst && (!m.available[2].Get() || layer == 2) {
			subBestQuality(layer)
		} else if !bestQualityFirst && (!m.available[0].Get() || layer == 0) {
			subLowestQuality(layer)
		}
	}

	go m.deliver(src, layer)
}

func (m *Multicast) isDownTrackSubscribed(layer int, dt IDownTrack) bool {
	dts := m.downTracks[layer].Load().([]IDownTrack)
	for _, cdt := range dts {
		if cdt == dt {
			return true
		}
	}
	return false
}

func (m *Multicast) AddDownTrack(track IDownTrack, bestQualityFirst bool) {
	if m.closed.Get() {
		return
	}

	layer := 0
	if m.isSimulcast {
		for i, t := range m.available {
			if t.Get() {
				layer = i
				if !bestQualityFirst {
					break
				}
			}
		}
		if m.isDownTrackSubscribed(layer, track) {
			return
		}
		track.SetInitialLayers(layer, 2)
		track.SetMaxSpatialLayer(2)
		track.SetMaxTemporalLayer(2)
		track.SetSimulcast(true)
	} else {
		if m.isDownTrackSubscribed(layer, track) {
			return
		}
		track.SetInitialLayers(0, 0)
		track.SetSimulcast(false)
	}
	m.mutex.Lock()
	m.storeDownTrack(layer, track)
	m.mutex.Unlock()
}

func (m *Multicast) storeDownTrack(layer int, dt IDownTrack) {
	dts := m.downTracks[layer].Load().([]IDownTrack)
	ndts := make([]IDownTrack, len(dts)+1)
	copy(ndts, dts)
	ndts[len(ndts)-1] = dt
	m.downTracks[layer].Store(ndts)
}

func (m *Multicast) deliver(src IFrameSource, layer int) {
	defer func() {
		m.closeOnce.Do(func() {
			m.closed.Set(true)
			m.closeTracks()
		})
	}()

	for {
		pkt, err := src.ReadFrame()
		if err == io.EOF {
			return
		}

		if m.isSimulcast {
			if m.pending[layer].Get() {
				if pkt.KeyFrame {
					m.mutex.Lock()
					for idx, dt := range m.pendingTracks[layer] {
						m.deleteDownTrack(dt.CurrentSpatialLayer(), dt.TrackID())
						m.storeDownTrack(layer, dt)
						dt.SwitchSpatialLayerDone(layer)
						m.pendingTracks[layer][idx] = nil
					}
					m.pendingTracks[layer] = m.pendingTracks[layer][:0]
					m.pending[layer].Set(false)
					m.mutex.Unlock()
				} else {
					if e := src.RequestKeyFrame(); e != nil {
						m.logger.WithError(e).Error("Error requesting key frame")
					}
				}
			}
		}

		for _, dt := range m.downTracks[layer].Load().([]IDownTrack) {
			pkt.Layer = int8(layer)
			if err = dt.WriteFrame(pkt); err != nil {
				if err == io.EOF || err == io.ErrClosedPipe {
					m.mutex.Lock()
					m.deleteDownTrack(layer, dt.TrackID())
					m.mutex.Unlock()
				}
				m.logger.WithError(err).Error("Error writing frame to down track")
			}
		}
	}
}

func (m *Multicast) closeTracks() {
	for idx, a := range m.available {
		if !a.Get() {
			continue
		}
		for _, dt := range m.downTracks[idx].Load().([]IDownTrack) {
			dt.Close()
		}
	}

	if m.onCloseHandler != nil {
		m.onCloseHandler()
	}
}

func (m *Multicast) DeleteDownTrack(layer int, id string) {
	if m.closed.Get() {
		return
	}
	m.mutex.Lock()
	m.deleteDownTrack(layer, id)
	m.mutex.Unlock()
}

func (m *Multicast) deleteDownTrack(layer int, trackID string) {
	dts := m.downTracks[layer].Load().([]IDownTrack)
	ndts := make([]IDownTrack, 0, len(dts))
	for _, dt := range dts {
		if dt.TrackID() != trackID {
			ndts = append(ndts, dt)
		} else {
			dt.Close()
		}
	}
	m.downTracks[layer].Store(ndts)
}

func (m *Multicast) OnClose(f func()) {
	m.onCloseHandler = f
}

func (m *Multicast) SwitchDownTrack(track IDownTrack, layer int) error {
	if m.closed.Get() {
		return errors.New("multicast closed")
	}

	if m.available[layer].Get() {
		m.mutex.Lock()
		m.pending[layer].Set(true)
		m.pendingTracks[layer] = append(m.pendingTracks[layer], track)
		m.mutex.Unlock()
		return nil
	}

	return errors.New("layer not available")
}

func (m *Multicast) FrameFormat() FrameFormat {
	return m.format
}

func (m *Multicast) PacketType() PacketType {
	return m.pt
}
