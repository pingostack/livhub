package forwarder

import "sync"

type IRouter interface {
	ID() string
	AddDownTrack(track IDownTrack) error
	AddUpTrack(track IUpTrack) error
}

type RouterConfig struct {
	DynamicFilter    bool
	BestQualityFirst bool
}

type Router struct {
	id         string
	multicasts map[string]*Multicast
	mutex      sync.RWMutex
	config     RouterConfig
}

func NewRouter(id string, config RouterConfig) IRouter {
	return &Router{
		id:         id,
		multicasts: make(map[string]*Multicast),
		config:     config,
	}
}

func (r *Router) ID() string {
	return r.id
}

func (r *Router) AddDownTrack(track IDownTrack) error {
	return nil
}

func (r *Router) AddUpTrack(track IUpTrack) error {
	if r.config.DynamicFilter {
		filters := DefaultFilterFactory.Filters()
		for _, filter := range filters {

		}
	}

	multicast := r.getMulticast(track.FrameFormat(), track.PacketType())
	if multicast == nil {
		multicast = NewMulticast(r.id, track.PacketType(), track.FrameFormat(), track.Simulcast())
		r.addMulticast(multicast)
	}
	multicast.AddUpTrack(track, track.Layer(), r.config.BestQualityFirst)

	return nil
}

func (r *Router) addMulticast(multicast *Multicast) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := multicast.FrameFormat().String() + "-" + multicast.PacketType().String()
	r.multicasts[id] = multicast
}

func (r *Router) removeMulticast(multicast *Multicast) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := multicast.FrameFormat().String() + "-" + multicast.PacketType().String()
	delete(r.multicasts, id)
}

func (r *Router) getMulticast(codecType FrameFormat, packetType PacketType) *Multicast {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	id := codecType.String() + "-" + packetType.String()
	return r.multicasts[id]
}
