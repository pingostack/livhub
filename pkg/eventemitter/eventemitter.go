package eventemitter

import (
	"context"
	"fmt"
	"sync"

	"github.com/pingostack/neon/pkg/logger"
	"go.uber.org/atomic"
)

type eventID int

type EventEmitter interface {
	AddEvent(eventID eventID, f func(data interface{}))
	EmitEvent(eventID eventID, data interface{}) error
}

type Event struct {
	Signal eventID
	Data   interface{}
}

func (e Event) String() string {
	return fmt.Sprintf("{eventID: %d, data: %+v}", e.Signal, e.Data)
}

type EventEmitterImpl struct {
	oneventLock sync.RWMutex
	eventCh     chan Event
	listeners   map[eventID][]func(data interface{})
	logger      logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
}

var (
	signalCounter atomic.Int32
)

func GenEventID() eventID {
	return eventID(signalCounter.Inc())
}

func NewEventEmitter(ctx context.Context, size int, logger logger.Logger) EventEmitter {
	m := &EventEmitterImpl{
		eventCh:   make(chan Event, size),
		listeners: make(map[eventID][]func(data interface{})),
		logger:    logger,
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	go m.run()

	return m
}

func (m *EventEmitterImpl) EmitEvent(eventID eventID, data interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Errorf("EventEmitter panic: %v", r)
			}
		}
	}()

	e := Event{
		Signal: eventID,
		Data:   data,
	}

	select {
	case m.eventCh <- e:
	default:
		if m.logger != nil {
			m.logger.Warnf("Event queue full, Event: %s", e)
		}
		return fmt.Errorf("Event queue full")
	}

	return nil
}

func (m *EventEmitterImpl) run() {
	defer func() {
		if m.logger != nil {
			m.logger.Infof("EventEmitter stopped")
		}
		close(m.eventCh)
	}()

	for {
		select {
		case <-m.ctx.Done():
			return

		case e, ok := <-m.eventCh:
			if !ok {
				return
			}

			m.oneventLock.RLock()
			listeners, found := m.listeners[e.Signal]
			m.oneventLock.RUnlock()

			if !found {
				continue
			}

			for _, f := range listeners {
				f(e.Data)
			}
		}
	}
}

func (m *EventEmitterImpl) AddEvent(eventID eventID, f func(data interface{})) {
	m.oneventLock.Lock()
	defer m.oneventLock.Unlock()

	listeners, found := m.listeners[eventID]
	if !found {
		listeners = make([]func(data interface{}), 0)
	}

	listeners = append(listeners, f)

	m.listeners[eventID] = listeners
}

func (m *EventEmitterImpl) Remove(eventID eventID) {
	m.oneventLock.Lock()
	defer m.oneventLock.Unlock()

	delete(m.listeners, eventID)
}

func (m *EventEmitterImpl) Close() {
	m.cancel()
}
