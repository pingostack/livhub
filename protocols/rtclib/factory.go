package rtclib

import (
	"context"

	"github.com/pingostack/neon/pkg/eventemitter"
	"github.com/pingostack/neon/pkg/logger"
	"github.com/pingostack/neon/protocols/rtclib/config"
	"github.com/pingostack/neon/protocols/rtclib/transport"
	"github.com/pkg/errors"
)

const (
	defaultEventEmitterLength = 10
)

type ProducerParams struct {
	AllowdCodecs []config.CodecConfig
	Ctx          context.Context
	Logger       logger.Logger
	PreferTCP    bool
}

type ConsumerParams struct {
	AllowdCodecs []config.CodecConfig
	Ctx          context.Context
	Logger       logger.Logger
	PreferTCP    bool
}

type Factory interface {
	NewProducer(params ProducerParams) (*Producer, error)
	NewConsumer(params ConsumerParams) (*Consumer, error)
}

type FactoryImpl struct {
	settings     config.Settings
	webrtcConfig *config.WebRTCConfig
}

func NewTransportFactory(settings config.Settings) Factory {
	f := &FactoryImpl{
		settings: settings,
	}

	if e := settings.Validate(); e != nil {
		panic(errors.Wrap(e, "invalid settings"))
	}

	webrtcConfig, err := config.NewWebRTCConfig(&settings)
	if err != nil {
		panic(errors.Wrap(err, "invalid webrtc config"))
	}

	f.webrtcConfig = webrtcConfig

	return f
}

func (f *FactoryImpl) NewProducer(params ProducerParams) (*Producer, error) {
	em := eventemitter.NewEventEmitter(params.Ctx, defaultEventEmitterLength, params.Logger)

	transport, err := transport.NewTransport(transport.WithWebRTCConfig(f.webrtcConfig),
		transport.WithConnConfig(&f.settings.ICEConfig),
		transport.WithAllowedCodecs(params.AllowdCodecs),
		transport.WithLogger(params.Logger),
		transport.WithContext(params.Ctx),
		transport.WithEventEmitter(em))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create transport")
	}

	transport.SetPreferTCP(params.PreferTCP)

	p, err := NewProducer(transport)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create producer")
	}

	return p, nil
}

func (f *FactoryImpl) NewConsumer(params ConsumerParams) (*Consumer, error) {
	em := eventemitter.NewEventEmitter(params.Ctx, defaultEventEmitterLength, params.Logger)

	transport, err := transport.NewTransport(transport.WithWebRTCConfig(f.webrtcConfig),
		transport.WithConnConfig(&f.settings.ICEConfig),
		transport.WithAllowedCodecs(params.AllowdCodecs),
		transport.WithLogger(params.Logger),
		transport.WithContext(params.Ctx),
		transport.WithEventEmitter(em))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create transport")
	}

	transport.SetPreferTCP(params.PreferTCP)

	c, err := NewConsumer(transport)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create consumer")
	}

	return c, nil
}
