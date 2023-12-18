package core

import (
	"sync"
)

type Namespace struct {
	name    string
	domains []string
	streams map[string]*Stream
	lock    sync.RWMutex
}

func NewNamespace(name string, domains ...string) *Namespace {
	return &Namespace{
		name:    name,
		domains: domains,
		streams: make(map[string]*Stream),
	}
}

func (ns *Namespace) Name() string {
	ns.lock.RLock()
	defer ns.lock.RUnlock()
	return ns.name
}

func (ns *Namespace) Domains() []string {
	ns.lock.RLock()
	defer ns.lock.RUnlock()
	domains := make([]string, len(ns.domains))
	copy(domains, ns.domains)
	return domains
}

func (ns *Namespace) Stream(name string) *Stream {
	ns.lock.RLock()
	defer ns.lock.RUnlock()
	return ns.streams[name]
}

func (ns *Namespace) GetOrNewStream(id string) (*Stream, bool) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	stream, ok := ns.streams[id]
	if !ok {
		stream = NewStream(id)
		ns.streams[id] = stream
	}
	return stream, !ok
}

func (ns *Namespace) DeleteStream(name string) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	delete(ns.streams, name)
}

func (ns *Namespace) Streams() []*Stream {
	ns.lock.RLock()
	defer ns.lock.RUnlock()
	streams := make([]*Stream, len(ns.streams))
	i := 0
	for _, stream := range ns.streams {
		streams[i] = stream
		i++
	}

	return streams
}

func (ns *Namespace) AddDomain(domains ...string) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	ns.domains = append(ns.domains, domains...)
}

func (ns *Namespace) DeleteDomain(domain string) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	for i, d := range ns.domains {
		if d == domain {
			ns.domains = append(ns.domains[:i], ns.domains[i+1:]...)
			break
		}
	}
}

func (ns *Namespace) HasDomain(domain string) bool {
	ns.lock.RLock()
	defer ns.lock.RUnlock()
	for _, d := range ns.domains {
		if d == domain {
			return true
		}
	}
	return false
}
