package event

import (
	"context"

	"github.com/let-light/gomodule"
	feature_event "github.com/pingostack/livhub/features/core"
	"github.com/pingostack/livhub/pkg/eventemitter"
	"github.com/sirupsen/logrus"
)

type Event struct {
	eventemitter.EventEmitter
}

func init() {
	event := &Event{
		EventEmitter: eventemitter.NewEventEmitter(context.Background(), 100, logrus.WithField("module", "event")),
	}
	gomodule.AddFeature(event)
}

func (e *Event) Type() interface{} {
	return feature_event.Type()
}
