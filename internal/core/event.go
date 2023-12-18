package core

import "context"

type eventInfo struct {
	name string
	args []interface{}
}

type eventemitter struct {
	chEvent   chan eventInfo
	listeners map[string][]func(...interface{})
	ctx       context.Context
}

var EventEmitter *eventemitter

func InitEventEmitter(ctx context.Context) {
	EventEmitter = &eventemitter{
		chEvent:   make(chan eventInfo, 100),
		listeners: make(map[string][]func(...interface{})),
		ctx:       ctx,
	}

	go EventEmitter.run()
}

func (e *eventemitter) run() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case event := <-e.chEvent:
			e.Emit(event.name, event.args...)
		}
	}
}

func (e *eventemitter) On(event string, listener func(...interface{})) {
	e.listeners[event] = append(e.listeners[event], listener)
}

func (e *eventemitter) Emit(event string, args ...interface{}) {
	if listeners, ok := e.listeners[event]; ok {
		for _, listener := range listeners {
			listener(args...)
		}
	}
}

func (e *eventemitter) EmitAsync(event string, args ...interface{}) {
	e.chEvent <- eventInfo{
		name: event,
		args: args,
	}
}
