package rtc

import (
	"context"
	"sync"

	"github.com/im-pingo/gomodule"
	feature_rtc "github.com/pingostack/neon/features/rtc"
	"github.com/pingostack/neon/pkg/rtclib"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rtcModule *rtc

type rtc struct {
	gomodule.DefaultModule
	ctx              context.Context
	preSettings      feature_rtc.Settings
	settings         *feature_rtc.Settings
	logger           *logrus.Entry
	lock             sync.RWMutex
	transportFactory rtclib.StreamFactory
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

func (rtc *rtc) Type() interface{} {
	return feature_rtc.Type()
}

func (rtc *rtc) GetSettings() feature_rtc.Settings {
	return *rtc.settings
}

func (rtc *rtc) StreamFactory() rtclib.StreamFactory {
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

func StreamFactory() rtclib.StreamFactory {
	return rtcModule.StreamFactory()
}
