package router

import (
	"sync"
)

type Namespace struct {
	name    string
	domains []string
	routers map[string]*Router
	lock    sync.RWMutex
}

func NewNamespace(name string, domains ...string) *Namespace {
	return &Namespace{
		name:    name,
		domains: domains,
		routers: make(map[string]*Router),
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

func (ns *Namespace) Router(name string) *Router {
	ns.lock.RLock()
	defer ns.lock.RUnlock()
	return ns.routers[name]
}

func (ns *Namespace) GetOrNewRouter(id string) (*Router, bool) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	router, ok := ns.routers[id]
	if !ok {
		router = NewRouter(ns, id)
		ns.routers[id] = router
	}
	return router, !ok
}

func (ns *Namespace) DeleteRouter(name string) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	delete(ns.routers, name)
}

func (ns *Namespace) Routers() []*Router {
	ns.lock.RLock()
	defer ns.lock.RUnlock()
	routers := make([]*Router, len(ns.routers))
	i := 0
	for _, router := range ns.routers {
		routers[i] = router
		i++
	}

	return routers
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
