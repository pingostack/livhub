package event

import (
	"context"

	"github.com/im-pingo/gomodule"
	feature_event "github.com/pingostack/neon/features/core"
	"github.com/pingostack/neon/pkg/eventemitter"
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
