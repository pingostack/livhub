package test

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingostack/livhub/pkg/plugo"
	"github.com/stretchr/testify/assert"
)

func TestLocalConfigurationManagement(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	initialConfig := []byte(`
test:
  name: "test-plugin"
  version: 1
`)

	err := os.WriteFile(configFile, initialConfig, 0644)
	assert.NoError(t, err, "Writing initial config should succeed")

	// Create test plugin
	plugin := &TestPlugin{
		config: &TestConfig{},
	}

	// Register plugin with config
	var configUpdated atomic.Bool
	err = plugo.RegisterPlugin("test",
		plugo.WithConfig("test", &TestConfig{}, func(ctx context.Context, pl plugo.Plugin, cfg interface{}) error {
			if c, ok := cfg.(*TestConfig); ok {
				plugin.config = c
				configUpdated.Store(true)
			} else {
				t.Logf("Config type assertion failed, got type: %T", cfg)
			}
			return nil
		}),
	)
	assert.NoError(t, err, "Plugin registration should succeed")

	// Create file provider
	provider, err := plugo.NewFileProvider(configFile)
	assert.NoError(t, err, "Creating file provider should succeed")

	// Set config provider
	err = plugo.SetConfigProvider(provider)
	assert.NoError(t, err, "Setting config provider should succeed")

	plugo.Start()
	// Verify initial config
	assert.Eventually(t, configUpdated.Load, time.Second, 10*time.Millisecond, "Config should be loaded")
	assert.Equal(t, "test-plugin", plugin.config.Name, "Initial config name should match")
	assert.Equal(t, 1, plugin.config.Version, "Initial config version should match")

	// Test config update
	configUpdated.Store(false)
	updatedConfig := []byte(`
test:
  name: "updated-plugin"
  version: 2
`)

	err = os.WriteFile(configFile, updatedConfig, 0644)
	assert.NoError(t, err, "Writing updated config should succeed")

	// Verify config update
	time.Sleep(1000 * time.Millisecond)
	assert.Eventually(t, configUpdated.Load, time.Second, 10*time.Millisecond, "Config should be updated")
	assert.Equal(t, "updated-plugin", plugin.config.Name, "Updated config name should match")
	assert.Equal(t, 2, plugin.config.Version, "Updated config version should match")
}
