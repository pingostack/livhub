package forwarder

import (
	"context"
	"fmt"
	"sync"

	"github.com/let-light/neon/pkg/utils"
	"github.com/lucsky/cuid"
	"github.com/sirupsen/logrus"
)

type IPublisher interface {
	//	OnConnect(f func(data string, publisher IPublisher))
	Close()
	// Marshal() string
	OnUpTrack(f func(track IUpTrack))
	// ITrack
	// ITransport
}

type ISubscriber interface {
	//	OnConnect(f func(data string, subscriber ISubscriber))
	Close()
	NegotiateTracks()
	OnNegotiateTracks(f func())
	// Marshal() string
	OnDownTrack(f func(track IDownTrack))
	// ITrack
	// ITransport
}

type IPeer interface {
	ID() string
	Group() IGroup
	Publisher() IPublisher
	Subscriber() ISubscriber
	Close()
}

type IGroupProvider interface {
	GetOrNewGroup(groupId string) IGroup
}

type IPublisherProvider interface {
	NewPublisher(ctx context.Context, id string, logger *logrus.Entry) (IPublisher, error)
}

type ISubscriberProvider interface {
	NewSubscriber(ctx context.Context, id string, logger *logrus.Entry) (ISubscriber, error)
}

type JoinConfig struct {
	NoPublish       bool
	NoSubscribe     bool
	NoAutoSubscribe bool
}

type Peer struct {
	sync.Mutex
	closed             utils.AtomicBool
	id                 string
	publisherProvider  IPublisherProvider
	subscriberProvider ISubscriberProvider
	groupProvider      IGroupProvider
	publisher          IPublisher
	subscriber         ISubscriber
	group              IGroup
	logger             *logrus.Entry
	OnConnect          func()
	OnRemoteTrack      func()
	OnLocalTrack       func()
	ctx                context.Context
	cancel             context.CancelFunc
	router             IRouter
}

func NewPeer(ctx context.Context, id string, logger *logrus.Entry,
	publisherProvider IPublisherProvider,
	subscriberProvider ISubscriberProvider,
	groupProvider IGroupProvider) *Peer {

	if id == "" {
		id = cuid.New()
	}

	p := &Peer{
		publisherProvider:  publisherProvider,
		subscriberProvider: subscriberProvider,
		groupProvider:      groupProvider,
		logger:             logger.WithFields(logrus.Fields{"package": "forwarder", "peer_id": id}),
		id:                 id,
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	p.Logger().Info("new peer created")

	return p
}

func (p *Peer) Logger() *logrus.Entry {
	return p.logger
}

func (p *Peer) Join(groupId string, config JoinConfig) error {
	p.Logger().Info("joining group")

	if p.closed.Get() {
		return fmt.Errorf("peer is closed")
	}

	p.group = p.groupProvider.GetOrNewGroup(groupId)
	if p.group == nil {
		return fmt.Errorf("failed to get group")
	}

	if p.publisherProvider != nil && !config.NoPublish {
		publisher, err := p.publisherProvider.NewPublisher(p.ctx, p.id, p.Logger())
		if err != nil {
			p.Logger().WithError(err).Error("failed to create publisher")
			return err
		}

		publisher.OnUpTrack(func(track IUpTrack) {
			p.Logger().WithField(
				"track_id", track.TrackID(),
			).Info("up track received")
			p.router = NewRouter(p.id)
			p.router.AddUpTrack(track)
		})

		p.publisher = publisher
	}

	if p.subscriberProvider != nil && !config.NoSubscribe {
		subscriber, err := p.subscriberProvider.NewSubscriber(p.ctx, p.id, p.Logger())
		if err != nil {
			p.Logger().WithError(err).Error("failed to create subscriber")
			return err
		}

		p.subscriber.OnDownTrack(func(track IDownTrack) {
			p.Logger().WithField(
				"track_id", track.TrackID(),
			).Info("down track received")
		})

		p.subscriber = subscriber
	}

	p.group.AddPeer(p)

	if !config.NoSubscribe {
		p.group.Subscribe(p)
	}

	return nil
}

func (p *Peer) ID() string {
	return p.id
}

func (p *Peer) Group() IGroup {
	return p.group
}

func (p *Peer) Publisher() IPublisher {
	return p.publisher
}

func (p *Peer) Subscriber() ISubscriber {
	return p.subscriber
}

func (p *Peer) Close() {
	p.Lock()
	defer p.Unlock()

	if !p.closed.Set(true) {
		return
	}

	if p.group != nil {
		p.group.RemovePeer(p)
	}

	if p.publisher != nil {
		p.publisher.Close()
	}
	if p.subscriber != nil {
		p.subscriber.Close()
	}
}
