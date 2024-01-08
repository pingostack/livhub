package router

import "sync"

type NSManager struct {
	namespaces map[string]*Namespace
	lock       sync.RWMutex
}

func NewNSManager() *NSManager {
	return &NSManager{
		namespaces: make(map[string]*Namespace),
	}
}

func (m *NSManager) LookupDomain(domain string) (*Namespace, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	ns, ok := m.namespaces[domain]
	return ns, ok
}

func (m *NSManager) GetOrNewNamespace(name string, domains ...string) *Namespace {
	m.lock.Lock()
	defer m.lock.Unlock()
	ns, ok := m.namespaces[name]
	if !ok {
		ns = NewNamespace(name, domains...)
		m.namespaces[name] = ns
	}

	return ns
}

func (m *NSManager) DeleteNamespace(name string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.namespaces, name)
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
