package feature_rtc

import (
	"github.com/let-light/gomodule"

	rtc_conf "github.com/pingostack/neon/pkg/rtclib/config"
)

type Settings struct {
	DefaultSettings rtc_conf.Settings `json:"default" mapstructure:"default" yaml:"default"`
}

type Feature interface {
	gomodule.IModule
	GetSettings() Settings
}

func Type() interface{} {
	return (*Feature)(nil)
}
