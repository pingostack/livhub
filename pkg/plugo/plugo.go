package plugo

import (
	"context"

	"github.com/pkg/errors"
)

var defaultPluginManager *pluginManager
var defaultFeatureManager *featureManager

func init() {
	defaultPluginManager = newManager(context.Background())
	defaultFeatureManager = newFeatureManager()
}

func RegisterPlugin(name string, options ...registerOption) error {
	return defaultPluginManager.register(name, options...)
}

func UnregisterPlugin(name string) {
	defaultPluginManager.unregister(name)
}

func Start() error {
	if err := defaultPluginManager.Start(); err != nil {
		return errors.Wrap(err, "failed to start plugin manager")
	}

	if err := defaultFeatureManager.Start(); err != nil {
		return errors.Wrap(err, "failed to start feature manager")
	}

	return nil
}

func Close() error {
	var errs []interface{}
	if err := defaultPluginManager.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := defaultFeatureManager.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Wrap(errors.New(Concat(errs...)), "failed to close plugo")
	}

	return nil
}

func SetConfigProvider(provider ConfigProvider) error {
	return defaultPluginManager.configLoader.SetProvider(provider)
}

func RequireFeatures(callback interface{}) error {
	return defaultFeatureManager.RequireFeatures(callback)
}

func AddFeature(feature Feature) {
	defaultFeatureManager.AddFeature(feature)
}
