package plugo

import (
	"context"
	"reflect"
)

type configInfo struct {
	configKey     string
	configType    reflect.Type
	Value         interface{}
	defaultValue  interface{}
	lastConfig    interface{} // Store the last successfully applied configuration
}

type plugInfo struct {
	name        string
	obj         interface{}
	featureType reflect.Type
	init        func(ctx context.Context) error
	pre         func(ctx context.Context) error
	run         func(ctx context.Context) error
	exit        func(ctx context.Context) error

	onConfigChange func(ctx context.Context, cfg interface{}) error
	config         configInfo
}
