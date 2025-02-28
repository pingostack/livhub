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
	name     string
	obj      Plugin
	create   func(ctx context.Context) Plugin
	setup    func(ctx context.Context, pl Plugin) error
	run      func(ctx context.Context, pl Plugin) error
	exit     func(ctx context.Context, pl Plugin) error
	critical bool // Flag to mark if the plugin is critical for the system

	setupComplete int32
	running       int32
	configLoaded  int32

	onConfigChange func(ctx context.Context, pl Plugin, cfg interface{}) error
	config         configInfo
}
