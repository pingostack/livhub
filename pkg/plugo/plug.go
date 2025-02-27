package plugo

import (
	"context"
	"reflect"
)

type Plugin interface{}

type configInfo struct {
	configKey    string
	configType   reflect.Type
	Value        interface{}
	defaultValue interface{}
	lastConfig   interface{} // Store the last successfully applied configuration
}

type plugInfo struct {
	name string
	obj  Plugin
	init func(ctx context.Context, pl Plugin) error
	pre  func(ctx context.Context, pl Plugin) error
	run  func(ctx context.Context, pl Plugin) error
	exit func(ctx context.Context, pl Plugin) error

	onConfigChange func(ctx context.Context, pl Plugin, cfg interface{}) error
	config         configInfo
}
