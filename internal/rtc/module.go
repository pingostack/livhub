package rtc

import (
	"context"
	"sync"

	"github.com/let-light/gomodule"
	"github.com/pingostack/neon/protocols/rtclib"
	rtc_conf "github.com/pingostack/neon/protocols/rtclib/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rtcModule *rtc

type rtcSettings struct {
	DefaultSettings rtc_conf.Settings `json:"default" mapstructure:"default" yaml:"default"`
}

type rtc struct {
	gomodule.DefaultModule
	ctx              context.Context
	preSettings      rtcSettings
	settings         *rtcSettings
	logger           *logrus.Entry
	lock             sync.RWMutex
	transportFactory rtclib.Factory
}

func init() {
	rtcModule = &rtc{
		logger: logrus.WithField("module", "webrtc"),
	}
}

func RtcModule() *rtc {
	return rtcModule
}

func (rtc *rtc) InitModule(ctx context.Context, _ *gomodule.Manager) (interface{}, error) {
	rtc.ctx = ctx
	return &rtc.preSettings, nil
}

func (rtc *rtc) InitCommand() ([]*cobra.Command, error) {

	return nil, nil
}

func (rtc *rtc) ConfigChanged() {
	if rtc.settings == nil {
		rtc.settings = &rtc.preSettings
		rtc.logger.WithField("settings", rtc.settings).Info("settings changed")
	}
}

func (rtc *rtc) ModuleRun() {
}

func (rtc *rtc) TransportFactory() rtclib.Factory {
	rtc.lock.RLock()
	factory := rtc.transportFactory
	if rtc.transportFactory == nil {
		rtc.lock.RUnlock()
		rtc.lock.Lock()
		if rtc.transportFactory == nil {
			factory = rtclib.NewTransportFactory(rtc.settings.DefaultSettings)
			rtc.transportFactory = factory
		}
		rtc.lock.Unlock()

		return factory
	}

	rtc.lock.RUnlock()
	return factory
}

func TransportFactory() rtclib.Factory {
	return rtcModule.TransportFactory()
}
