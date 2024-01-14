package router

import (
	"context"
	"sync"
)

type NSManagerParams struct {
	Namespaces             map[string]NamespaceParams `yaml:"namespaces" json:"namespaces" mapstructure:"namespaces"`
	DefaultNamespaceParams NamespaceParams            `yaml:"default_namespace" json:"default_namespace" mapstructure:"default_namespace"`
}

type NSManager struct {
	namespaces map[string]*Namespace
	lock       sync.RWMutex
	params     NSManagerParams
}

func NewNSManager(params NSManagerParams) *NSManager {
	return &NSManager{
		namespaces: make(map[string]*Namespace),
		params:     params,
	}
}

func (m *NSManager) LookupDomain(domain string) (*Namespace, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	ns, ok := m.namespaces[domain]
	return ns, ok
}

func (m *NSManager) GetOrNewNamespace(ctx context.Context, name string) (*Namespace, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	ns, ok := m.namespaces[name]
	if !ok {
		var params NamespaceParams
		if params, ok = m.params.Namespaces[name]; !ok {
			params = m.params.DefaultNamespaceParams
		}

		if len(params.Domains) == 0 {
			params.Domains = []string{name}
		}
		ns = NewNamespace(ctx, params)
		m.namespaces[name] = ns
	}

	return ns, ok
}

func (m *NSManager) Namespaces() []*Namespace {
	m.lock.RLock()
	defer m.lock.RUnlock()
	namespaces := make([]*Namespace, 0, len(m.namespaces))
	for _, ns := range m.namespaces {
		namespaces = append(namespaces, ns)
	}
	return namespaces
}

func (m *NSManager) String() string {
	return "NSManager" // TODO
}
