package rtclib

import (
	"context"

	"github.com/pingostack/neon/pkg/eventemitter"
	"github.com/pingostack/neon/pkg/logger"
	"github.com/pingostack/neon/pkg/rtclib/config"
	"github.com/pingostack/neon/pkg/rtclib/transport"
	"github.com/pkg/errors"
)

const (
	defaultEventEmitterLength = 10
)

type RemoteStreamParams struct {
	AllowdCodecs []config.CodecConfig
	Ctx          context.Context
	Logger       logger.Logger
	PreferTCP    bool
}

type LocalStreamParams struct {
	AllowdCodecs []config.CodecConfig
	Ctx          context.Context
	Logger       logger.Logger
	PreferTCP    bool
}

type StreamFactory interface {
	NewRemoteStream(params RemoteStreamParams) (*RemoteStream, error)
	NewLocalStream(params LocalStreamParams) (*LocalStream, error)
}

type FactoryImpl struct {
	settings     config.Settings
	webrtcConfig *config.WebRTCConfig
}

func NewTransportFactory(settings config.Settings) StreamFactory {
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

func (f *FactoryImpl) NewRemoteStream(params RemoteStreamParams) (*RemoteStream, error) {
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

	p, err := NewRemoteStream(transport)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create remote stream")
	}

	return p, nil
}

func (f *FactoryImpl) NewLocalStream(params LocalStreamParams) (*LocalStream, error) {
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

	c, err := NewLocalStream(transport)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create local stream")
	}

	return c, nil
}
