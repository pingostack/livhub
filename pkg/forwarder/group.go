package forwarder

import (
	"sync"

	"github.com/let-light/neon/pkg/utils"
	"github.com/sirupsen/logrus"
)

type IGroup interface {
	ID() string
	Publish(upTrack IFrameSource, layer int8)
	AddPeer(peer IPeer)
	GetPeer(peerID string) IPeer
	RemovePeer(peer IPeer)
	Subscribe(p IPeer)
}

type Group struct {
	gid            string
	peersMutex     sync.RWMutex
	peers          map[string]IPeer
	routersMutex   sync.RWMutex
	routers        map[string]IRouter
	closed         utils.AtomicBool
	onCloseHandler func()
	logger         *logrus.Entry
}

func NewGroup(gid string) IGroup {
	g := &Group{
		gid:    gid,
		peers:  make(map[string]IPeer),
		logger: logrus.WithFields(logrus.Fields{"package": "forwarder", "role": "group", "group_id": gid}),
	}

	return g
}

func (g *Group) ID() string {
	return g.gid
}

func (g *Group) AddPeer(peer IPeer) {
	g.peersMutex.Lock()
	defer g.peersMutex.Unlock()

	g.peers[peer.ID()] = peer
}

func (g *Group) GetPeer(peerID string) IPeer {
	g.peersMutex.RLock()
	defer g.peersMutex.RUnlock()

	return g.peers[peerID]
}

func (g *Group) RemovePeer(peer IPeer) {
	peerId := peer.ID()
	g.logger.WithFields(logrus.Fields{
		"peer_id": peerId,
	}).Info("RemovePeer")

	g.peersMutex.Lock()
	if g.peers[peerId] == peer {
		delete(g.peers, peerId)
	}
	peerCount := len(g.peers)
	g.peersMutex.Unlock()

	// Close group if no peers
	if peerCount == 0 {
		g.Close()
	}
}

func (g *Group) getRouter(routerID string) IRouter {
	g.peersMutex.RLock()
	defer g.peersMutex.RUnlock()

	return g.routers[routerID]
}

func (g *Group) GetOrNewRouter(routerID string) IRouter {
	g.routersMutex.Lock()
	defer g.routersMutex.Unlock()

	if router, ok := g.routers[routerID]; ok {
		return router
	}

	router := NewRouter(routerID, g)
	g.routers[routerID] = router

	return router
}

func (g *Group) Publish(upTrack IUpTrack) {
	if g.closed.Get() {
		g.logger.WithFields(logrus.Fields{
			"up_track_id": upTrack.TrackID(),
		}).Error("Publish failed: group closed")
		return
	}

	router := g.GetOrNewRouter(upTrack.StreamID())

	g.logger.WithFields(logrus.Fields{
		"up_track_id": upTrack.TrackID(),
	}).Info("Publish")

	for _, p := range g.Peers() {
		if router.ID() == p.ID() || p.Subscriber() == nil {
			continue
		}

		g.logger.WithFields(logrus.Fields{
			"router_id": router.ID(),
			"peer_id":   p.ID(),
		}).Info("Publish")

		if e := router.AddDownTracks(p.Subscriber()); e != nil {
			g.logger.WithFields(logrus.Fields{
				"router_id": router.ID(),
				"peer_id":   p.ID(),
			}).Error("Publish failed")
			continue
		}
	}
}

func (g *Group) Peers() []IPeer {
	g.peersMutex.RLock()
	defer g.peersMutex.RUnlock()
	peers := make([]IPeer, 0, len(g.peers))
	for _, peer := range g.peers {
		peers = append(peers, peer)
	}
	return peers
}

func (g *Group) Subscribe(peer IPeer) {
	g.peersMutex.Lock()
	peers := make([]IPeer, 0, len(g.peers))
	for _, p := range g.peers {
		if p == peer || p.Publisher() == nil {
			continue
		}
		peers = append(peers, p)
	}
	g.peersMutex.Unlock()

	for _, p := range peers {
		g.logger.WithFields(logrus.Fields{
			"peer_id": p.ID(),
		}).Info("Subscribe")

		if e := p.Publisher().GetRouter().AddDownTracks(peer.Subscriber()); e != nil {
			g.logger.WithFields(logrus.Fields{
				"peer_id": p.ID(),
			}).Error("Subscribe failed")
			continue
		}
	}

	peer.Subscriber().NegotiateTracks()
}

// OnClose is called when the group is closed
func (g *Group) OnClose(f func()) {
	g.onCloseHandler = f
}

func (g *Group) Close() {
	if !g.closed.Set(true) {
		return
	}

	if g.onCloseHandler != nil {
		g.onCloseHandler()
	}
}
