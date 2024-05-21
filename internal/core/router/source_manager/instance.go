package sourcemanager

import (
	"sync"

	"github.com/pingostack/neon/pkg/deliver"
)

type Instance struct {
	sources       map[string]deliver.FrameSource
	defaultSource deliver.FrameSource
	lock          sync.RWMutex
}

func NewInstance() *Instance {
	return &Instance{
		sources: make(map[string]deliver.FrameSource),
	}
}

func (i *Instance) AddSource(id string, source deliver.FrameSource) {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.sources[id] = source
	i.defaultSource = source
}

func (i *Instance) GetSource(id string) deliver.FrameSource {
	i.lock.RLock()
	defer i.lock.RUnlock()

	return i.sources[id]
}

func (i *Instance) RemoveSource(id string) {
	i.lock.Lock()
	defer i.lock.Unlock()

	delete(i.sources, id)
	if i.defaultSource.Metadata().GUID == id {
		i.defaultSource = nil
	}

	if len(i.sources) > 0 {
		for _, source := range i.sources {
			i.defaultSource = source
			break
		}
	}
}

func (i *Instance) Close() {
	i.lock.Lock()
	defer i.lock.Unlock()

	for _, source := range i.sources {
		source.Close()
	}
	i.sources = make(map[string]deliver.FrameSource)
}

func (i *Instance) Sources() map[string]deliver.FrameSource {
	i.lock.RLock()
	defer i.lock.RUnlock()

	return i.sources
}

func (i *Instance) SourceIDs() []string {
	i.lock.RLock()
	defer i.lock.RUnlock()

	ids := make([]string, 0, len(i.sources))
	for id := range i.sources {
		ids = append(ids, id)
	}

	return ids
}

func (i *Instance) AddIfNotExist(source deliver.FrameSource) bool {
	i.lock.Lock()
	defer i.lock.Unlock()

	if _, ok := i.sources[source.Metadata().GUID]; ok {
		return false
	}

	i.sources[source.Metadata().GUID] = source
	i.defaultSource = source

	return true
}

func (i *Instance) DefaultSource() deliver.FrameSource {
	i.lock.RLock()
	defer i.lock.RUnlock()

	return i.defaultSource
}
