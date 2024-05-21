package router

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

type RouterParams struct {
	IdleSubscriberTimeout int `yaml:"idle_subscriber_timeout" json:"idle_subscriber_timeout" mapstructure:"idle_subscriber_timeout"`
	MaxProducerTimeout    int `yaml:"max_producer_timeout" json:"max_producer_timeout" mapstructure:"max_producer_timeout"`
	MaxSubscriberTimeout  int `yaml:"max_subscriber_timeout" json:"max_subscriber_timeout" mapstructure:"max_subscriber_timeout"`
}

type NamespaceParams struct {
	Name                string                  `yaml:"name" json:"name" mapstructure:"name"`
	Domains             []string                `yaml:"domains" json:"domains" mapstructure:"domains"`
	DefaultRouterParams RouterParams            `yaml:"default_router" json:"default_router" mapstructure:"default_router"`
	RoutersParams       map[string]RouterParams `yaml:"routers" json:"routers" mapstructure:"routers"`
}

type Namespace struct {
	name    string
	domains []string
	routers map[string]Router
	lock    sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	logger  *logrus.Entry
	params  NamespaceParams
}

func NewNamespace(ctx context.Context, params NamespaceParams) *Namespace {
	ns := &Namespace{
		params:  params,
		name:    params.Name,
		domains: params.Domains,
		routers: make(map[string]Router),
		logger:  logrus.WithField("namespace", params.Name),
	}

	ns.ctx, ns.cancel = context.WithCancel(ctx)

	ns.logger.WithField("params", params).Debugf("namespace created")

	return ns
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

func (ns *Namespace) Router(name string) Router {
	ns.lock.RLock()
	defer ns.lock.RUnlock()
	return ns.routers[name]
}

func (ns *Namespace) GetOrNewRouter(id string) (Router, bool) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	router, ok := ns.routers[id]
	if !ok || router.Closed() {
		if ok {
			delete(ns.routers, id)
		}

		routerParams := ns.params.DefaultRouterParams
		if params, ok := ns.params.RoutersParams[id]; ok {
			routerParams = params
		}

		router = NewRouter(ns.ctx, ns, routerParams, id, ns.logger)
		ns.routers[id] = router
		go ns.waitRouterDone(router)
	}

	return router, !ok
}

func (ns *Namespace) waitRouterDone(router Router) {
	<-router.Context().Done()
	ns.lock.Lock()
	defer ns.lock.Unlock()

	delete(ns.routers, router.ID())
	ns.logger.Infof("router %s removed", router.ID())
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

func (ns *Namespace) RemoveRouter(r Router) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	checkRouter, ok := ns.routers[r.ID()]
	if !ok || checkRouter != r {
		return
	}

	delete(ns.routers, r.ID())
}
