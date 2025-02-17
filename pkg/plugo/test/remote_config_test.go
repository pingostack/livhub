package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingostack/livhub/pkg/plugo"
	"github.com/stretchr/testify/assert"
)

func TestRemoteConfigProvider(t *testing.T) {
	// Create a test server to simulate remote config
	var currentConfig atomic.Value
	currentConfig.Store(`
test:
  name: "remote-plugin"
  node_id: "yaml-node-id-1"
  version: 3
`)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(currentConfig.Load().(string)))
	}))
	defer ts.Close()

	// Create test plugin
	plugin := &TestPlugin{
		config: &TestConfig{},
	}

	// Register plugin with config
	var configUpdated atomic.Bool
	err := plugo.RegisterPlugin("test",
		plugo.WithFeatureType(plugin),
		plugo.WithConfig("test", &TestConfig{}, func(ctx context.Context, cfg interface{}) error {
			if c, ok := cfg.(*TestConfig); ok {
				plugin.config = c
				configUpdated.Store(true)
			}
			return nil
		}),
	)
	assert.NoError(t, err, "Plugin registration should succeed")

	// Create HTTP provider with YAML format and short poll interval
	pollInterval := time.Millisecond * 100
	provider, err := plugo.NewHTTPProvider(ts.URL, &plugo.RemoteConfigOptions{
		PollInterval: pollInterval,
		Format:       "yaml",
	})
	assert.NoError(t, err, "Creating HTTP provider should succeed")
	defer provider.Close()

	// Set config provider
	err = plugo.SetConfigProvider(provider)
	assert.NoError(t, err, "Setting config provider should succeed")

	// Verify initial config is loaded
	assert.Eventually(t, configUpdated.Load, time.Second, 10*time.Millisecond, "Initial config should be loaded")
	assert.Equal(t, "remote-plugin", plugin.config.Name, "Initial config name should match")
	assert.Equal(t, 3, plugin.config.Version, "Initial config version should match")
	assert.Equal(t, "yaml-node-id-1", plugin.config.NodeID, "Initial config node ID should match")

	// Reset the configUpdated flag
	configUpdated.Store(false)

	// Update the config on the server
	currentConfig.Store(`
test:
  name: "remote-plugin-updated"
  node_id: "yaml-node-id-2"
  version: 4
`)

	// Wait for the new config to be loaded (should happen within 2 poll intervals)
	assert.Eventually(t, configUpdated.Load, pollInterval*3, pollInterval/2, "Updated config should be loaded")
	assert.Equal(t, "remote-plugin-updated", plugin.config.Name, "Updated config name should match")
	assert.Equal(t, 4, plugin.config.Version, "Updated config version should match")
	assert.Equal(t, "yaml-node-id-2", plugin.config.NodeID, "Updated config node ID should match")
}

func TestRemoteConfigProviderJSON(t *testing.T) {
	// Create an atomic value to track config version
	var currentConfig atomic.Value
	currentConfig.Store(`{
		"test": {
			"name": "remote-plugin-json",
			"node_id": "json-node-id-1",
			"version": 4
		}
	}`)

	// Create a test server to simulate remote config with JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(currentConfig.Load().(string)))
	}))
	defer ts.Close()

	// Create test plugin
	plugin := &TestPlugin{
		config: &TestConfig{},
	}

	// Register plugin with config
	var configUpdated atomic.Bool
	err := plugo.RegisterPlugin("test",
		plugo.WithFeatureType(plugin),
		plugo.WithConfig("test", &TestConfig{}, func(ctx context.Context, cfg interface{}) error {
			if c, ok := cfg.(*TestConfig); ok {
				plugin.config = c
				configUpdated.Store(true)
			}
			return nil
		}),
	)
	assert.NoError(t, err, "Plugin registration should succeed")

	// Create HTTP provider with JSON format and short poll interval
	pollInterval := time.Millisecond * 100
	provider, err := plugo.NewHTTPProvider(ts.URL, &plugo.RemoteConfigOptions{
		PollInterval: pollInterval,
		Format:       "json",
	})
	assert.NoError(t, err, "Creating HTTP provider should succeed")
	defer provider.Close()

	// Set config provider
	err = plugo.SetConfigProvider(provider)
	assert.NoError(t, err, "Setting config provider should succeed")

	// Verify initial config is loaded
	assert.Eventually(t, configUpdated.Load, time.Second, 10*time.Millisecond, "Initial config should be loaded")
	assert.Equal(t, "remote-plugin-json", plugin.config.Name, "Initial config name should match")
	assert.Equal(t, 4, plugin.config.Version, "Initial config version should match")
	assert.Equal(t, "json-node-id-1", plugin.config.NodeID, "Initial config node ID should match")

	// Reset the configUpdated flag
	configUpdated.Store(false)
	// Update the config on the server
	currentConfig.Store(`{
		"test": {
			"name": "remote-plugin-json-updated",
			"node_id": "json-node-id-2",
			"version": 5
		}
	}`)

	// Wait for the new config to be loaded (should happen within 2 poll intervals)
	assert.Eventually(t, configUpdated.Load, pollInterval*3, pollInterval/2, "Updated config should be loaded")
	assert.Equal(t, "remote-plugin-json-updated", plugin.config.Name, "Updated config name should match")
	assert.Equal(t, 5, plugin.config.Version, "Updated config version should match")
	assert.Equal(t, "json-node-id-2", plugin.config.NodeID, "Updated config node ID should match")
}
