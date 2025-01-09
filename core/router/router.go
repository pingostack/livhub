package router

import (
	"context"
	"fmt"
	"time"

	"github.com/pingostack/livhub/core/peer"
	"github.com/pingostack/livhub/core/stream"
	"github.com/pingostack/livhub/pkg/avframe"
	"github.com/pingostack/livhub/pkg/errcode"
	"github.com/pingostack/livhub/pkg/logger"
)

type Router struct {
	id     string
	stream *stream.Stream
	ctx    context.Context
	cancel context.CancelFunc
	logger logger.Logger
}

func NewRouter(ctx context.Context, id string) *Router {
	ctx, cancel := context.WithCancel(ctx)

	r := &Router{
		id:     id,
		stream: stream.NewStream(ctx, id),
		ctx:    ctx,
		cancel: cancel,
		logger: logger.WithFields(map[string]interface{}{"router": id}),
	}

	go r.run()

	return r
}

func (r *Router) ID() string {
	return r.id
}

func (r *Router) Stream() *stream.Stream {
	return r.stream
}

func (r *Router) Publish(publisher peer.Publisher) error {
	return BuildPublishMiddleware(r.ctx, func(ctx context.Context, publisher peer.Publisher, stage Stage) error {
		err := r.stream.Publish(publisher)

		if err != nil {
			return err
		}

		return nil
	})(r.ctx, publisher, StageStart)
}

func (r *Router) Unpublish(publisher peer.Publisher) error {
	return BuildPublishMiddleware(r.ctx, func(ctx context.Context, publisher peer.Publisher, _ Stage) error {
		return r.stream.Unpublish(publisher)
	})(r.ctx, publisher, StageEnd)
}

func (r *Router) Subscribe(subscriber peer.Subscriber) error {
	return BuildSubscribeMiddleware(r.ctx, func(ctx context.Context, subscriber peer.Subscriber, _ Stage) error {
		processor, err := r.stream.Subscribe(subscriber, func(sub peer.Subscriber, processor avframe.Processor, err error) {
			if err != nil {
				return
			}
			sub.OnActive(processor)
		})

		if err != nil {
			if !errcode.Is(err, errcode.ErrPublisherNotSet) {
				return err
			}

			r.logger.WithField("subscriber", subscriber).Info("subscriber not set")

			return nil
		}

		subscriber.OnActive(processor)

		return nil
	})(r.ctx, subscriber, StageStart)
}

func (r *Router) Unsubscribe(subscriber peer.Subscriber) error {
	return BuildSubscribeMiddleware(r.ctx, func(ctx context.Context, subscriber peer.Subscriber, _ Stage) error {
		return r.stream.Unsubscribe(subscriber)
	})(r.ctx, subscriber, StageEnd)
}

func (r *Router) Close() {
	r.cancel()
}

func (r *Router) Done() <-chan struct{} {
	return r.ctx.Done()
}

func (r *Router) destory() {
	r.logger.Info("router destory")

	BuildStreamMiddleware(r.ctx, func(ctx context.Context, s *stream.Stream, stage Stage) error {
		return nil
	})(r.ctx, r.stream, StageEnd)
}

func (r *Router) run() {
	defer func() {
		if re := recover(); re != nil {
			r.logger.WithError(fmt.Errorf("panic: %v", re)).Error("router run panic")
		}

		// check if r.ctx is closed
		if r.ctx.Err() == nil {
			r.cancel()
		}

		r.destory()
	}()

	BuildStreamMiddleware(r.ctx, func(ctx context.Context, s *stream.Stream, stage Stage) error {
		return nil
	})(r.ctx, r.stream, StageStart)

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.stream.Done():
			return
		case <-time.After(1 * time.Second):
			r.logger.Debug("router alive")
		}
	}
}
