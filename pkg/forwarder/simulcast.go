package forwarder

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/let-light/neon/pkg/utils"
	"github.com/sirupsen/logrus"
)

type Simulcast struct {
	upTracks       [3]IFrameSource
	available      [3]utils.AtomicBool
	downTracks     [3]atomic.Value // []IDownTrack
	pending        [3]utils.AtomicBool
	pendingTracks  [3][]IDownTrack
	closed         utils.AtomicBool
	mutex          sync.Mutex
	peerID         string
	pt             PacketType
	format         FrameFormat
	closeOnce      sync.Once
	logger         *logrus.Entry
	filter         IFilter
	onCloseHandler func()
}

func NewSimulcast(peerID string, pt PacketType, format FrameFormat) *Simulcast {
	return &Simulcast{
		peerID: peerID,
		pt:     pt,
		format: format,
		logger: logrus.WithFields(logrus.Fields{"package": "forwarder", "role": "simulcast", "peer_id": peerID}),
	}
}

func (s *Simulcast) AddFilter(filter IFilter) {
	s.filter = filter
}

func (s *Simulcast) AddUpTrack(src IFrameSource, layer int, bestQualityFirst bool) {
	if s.closed.Get() {
		return
	}

	s.mutex.Lock()
	s.upTracks[layer] = src
	s.available[layer].Set(true)
	s.downTracks[layer].Store(make([]IDownTrack, 0, 10))
	s.pendingTracks[layer] = make([]IDownTrack, 0, 10)
	s.mutex.Unlock()

	subBestQuality := func(targetLayer int) {
		for l := 0; l < targetLayer; l++ {
			dts := s.downTracks[l].Load()
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
			dts := s.downTracks[l].Load()
			if dts == nil {
				continue
			}
			for _, dt := range dts.([]IDownTrack) {
				_ = dt.SwitchSpatialLayer(targetLayer, false)
			}
		}
	}

	if bestQualityFirst && (!s.available[2].Get() || layer == 2) {
		subBestQuality(layer)
	} else if !bestQualityFirst && (!s.available[0].Get() || layer == 0) {
		subLowestQuality(layer)
	}
}

func (s *Simulcast) isDownTrackSubscribed(layer int, dt IDownTrack) bool {
	dts := s.downTracks[layer].Load().([]IDownTrack)
	for _, cdt := range dts {
		if cdt == dt {
			return true
		}
	}
	return false
}

func (s *Simulcast) AddDownTrack(track IDownTrack, bestQualityFirst bool) {
	if s.closed.Get() {
		return
	}

	layer := 0
	for i, t := range s.available {
		if t.Get() {
			layer = i
			if !bestQualityFirst {
				break
			}
		}
	}
	if s.isDownTrackSubscribed(layer, track) {
		return
	}
	track.SetInitialLayers(layer, 2)
	track.SetMaxSpatialLayer(2)
	track.SetMaxTemporalLayer(2)
	track.SetSimulcast(true)

	s.mutex.Lock()
	s.storeDownTrack(layer, track)
	s.mutex.Unlock()
}

func (s *Simulcast) storeDownTrack(layer int, dt IDownTrack) {
	dts := s.downTracks[layer].Load().([]IDownTrack)
	ndts := make([]IDownTrack, len(dts)+1)
	copy(ndts, dts)
	ndts[len(ndts)-1] = dt
	s.downTracks[layer].Store(ndts)
}

func (s *Simulcast) WriteFrame(frame *Frame) error {
	defer func() {
		s.closeOnce.Do(func() {
			s.closed.Set(true)
			s.closeTracks()
		})
	}()

	layer := int(frame.Layer)
	if layer < 0 || layer > 2 {
		return errors.New("invalid layer")
	}

	if s.pending[layer].Get() {
		if frame.VFSI.IsKeyFrame {
			s.mutex.Lock()
			for idx, dt := range s.pendingTracks[layer] {
				s.deleteDownTrack(dt.CurrentSpatialLayer(), dt.TrackID())
				s.storeDownTrack(layer, dt)
				dt.SwitchSpatialLayerDone(layer)
				s.pendingTracks[layer][idx] = nil
			}
			s.pendingTracks[layer] = s.pendingTracks[layer][:0]
			s.pending[layer].Set(false)
			s.mutex.Unlock()
		} else {
			s.upTracks[layer].WriteFeedback(&FeedbackMsg{Type: FeedbackTypePLI})
		}
	}

	for _, dt := range s.downTracks[layer].Load().([]IDownTrack) {
		frame.Layer = int8(layer)
		if e := dt.WriteFrame(frame); e != nil {
			if e == io.EOF || e == io.ErrClosedPipe {
				s.mutex.Lock()
				s.deleteDownTrack(layer, dt.TrackID())
				s.mutex.Unlock()
			}
			s.logger.WithError(e).Error("Error writing frame to down track")
		}
	}

	return nil
}

func (s *Simulcast) closeTracks() {
	for idx, a := range s.available {
		if !a.Get() {
			continue
		}
		for _, dt := range s.downTracks[idx].Load().([]IDownTrack) {
			dt.Close()
		}
	}

	if s.onCloseHandler != nil {
		s.onCloseHandler()
	}
}

func (s *Simulcast) DeleteDownTrack(layer int, id string) {
	if s.closed.Get() {
		return
	}
	s.mutex.Lock()
	s.deleteDownTrack(layer, id)
	s.mutex.Unlock()
}

func (s *Simulcast) deleteDownTrack(layer int, trackID string) {
	dts := s.downTracks[layer].Load().([]IDownTrack)
	ndts := make([]IDownTrack, 0, len(dts))
	for _, dt := range dts {
		if dt.TrackID() != trackID {
			ndts = append(ndts, dt)
		} else {
			dt.Close()
		}
	}
	s.downTracks[layer].Store(ndts)
}

func (s *Simulcast) OnClose(f func()) {
	s.onCloseHandler = f
}

func (s *Simulcast) SwitchDownTrack(track IDownTrack, layer int) error {
	if s.closed.Get() {
		return errors.New("simulcast closed")
	}

	if s.available[layer].Get() {
		s.mutex.Lock()
		s.pending[layer].Set(true)
		s.pendingTracks[layer] = append(s.pendingTracks[layer], track)
		s.mutex.Unlock()
		return nil
	}

	return errors.New("layer not available")
}

func (s *Simulcast) FrameFormat() FrameFormat {
	return s.format
}

func (s *Simulcast) PacketType() PacketType {
	return s.pt
}
