package eventemitter

import (
	"context"
	"fmt"
	"sync"

	"github.com/pingostack/neon/pkg/logger"
	"go.uber.org/atomic"
)

type eventId int

type Event struct {
	Signal eventId
	Data   interface{}
}

func (e Event) String() string {
	return fmt.Sprintf("{eventId: %d, data: %+v}", e.Signal, e.Data)
}

type EventEmitter struct {
	oneventLock sync.RWMutex
	eventCh     chan Event
	listeners   map[eventId][]func(data interface{})
	logger      logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
}

var (
	signalCounter atomic.Int32
)

func GenEventId() eventId {
	return eventId(signalCounter.Inc())
}

func NewEventEmitter(ctx context.Context, size int, logger logger.Logger) *EventEmitter {
	m := &EventEmitter{
		eventCh:   make(chan Event, size),
		listeners: make(map[eventId][]func(data interface{})),
		logger:    logger,
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	go m.run()

	return m
}

func (m *EventEmitter) Emit(eventId eventId, data interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Errorf("EventEmitter panic: %v", r)
			}
		}
	}()

	e := Event{
		Signal: eventId,
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

func (m *EventEmitter) run() {
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

func (m *EventEmitter) Add(eventId eventId, f func(data interface{})) {
	m.oneventLock.Lock()
	defer m.oneventLock.Unlock()

	listeners, found := m.listeners[eventId]
	if !found {
		listeners = make([]func(data interface{}), 0)
	}

	listeners = append(listeners, f)

	m.listeners[eventId] = listeners
}

func (m *EventEmitter) Remove(eventId eventId) {
	m.oneventLock.Lock()
	defer m.oneventLock.Unlock()

	delete(m.listeners, eventId)
}

func (m *EventEmitter) Close() {
	m.cancel()
}
