package test

import (
	"context"
	"testing"

	"github.com/pingostack/livhub/pkg/plugo"
	"github.com/stretchr/testify/assert"
)

func TestPluginLifecycle(t *testing.T) {
	plugin := &TestPlugin{
		config: &TestConfig{},
	}

	// Test plugin registration
	err := plugo.RegisterPlugin("test",
		plugo.WithFeatureType(plugin),
		plugo.WithInit(func(ctx context.Context) error {
			plugin.initCalled.Store(true)
			return nil
		}),
		plugo.WithPre(func(ctx context.Context) error {
			plugin.preCalled.Store(true)
			return nil
		}),
		plugo.WithRun(func(ctx context.Context) error {
			plugin.runCalled.Store(true)
			return nil
		}),
		plugo.WithExit(func(ctx context.Context) error {
			plugin.exitCalled.Store(true)
			return nil
		}),
		plugo.WithConfig("test", &TestConfig{}, func(ctx context.Context, cfg interface{}) error {
			if c, ok := cfg.(*TestConfig); ok {
				plugin.config = c
			}
			return nil
		}),
	)

	assert.NoError(t, err, "Plugin registration should succeed")

	// Test initialization phase
	err = plugo.Start()
	assert.NoError(t, err, "Plugin initialization should succeed")
	assert.True(t, plugin.initCalled.Load(), "Init should be called")

	// Test pre-run phase
	assert.NoError(t, err, "Plugin pre-run should succeed")
	assert.True(t, plugin.preCalled.Load(), "Pre should be called")

	// Test run phase
	assert.NoError(t, err, "Plugin run should succeed")
	assert.True(t, plugin.runCalled.Load(), "Run should be called")

	// Test exit phase
	err = plugo.Close()
	assert.NoError(t, err, "Plugin exit should succeed")
	assert.True(t, plugin.exitCalled.Load(), "Exit should be called")
}
