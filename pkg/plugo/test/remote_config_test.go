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
	// 创建一个测试服务器来模拟远程配置
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := `
test:
  name: "remote-plugin"
  version: 3
`
		w.Write([]byte(config))
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

	// Create HTTP provider
	provider, err := plugo.NewHTTPProvider(ts.URL, "yaml", &plugo.RemoteConfigOptions{
		PollInterval: time.Millisecond * 100,
	})
	assert.NoError(t, err, "Creating HTTP provider should succeed")

	// Set config provider
	err = plugo.SetConfigProvider(provider)
	assert.NoError(t, err, "Setting config provider should succeed")

	// Verify remote config
	assert.Eventually(t, configUpdated.Load, time.Second, 10*time.Millisecond, "Config should be loaded")
	assert.Equal(t, "remote-plugin", plugin.config.Name, "Remote config name should match")
	assert.Equal(t, 3, plugin.config.Version, "Remote config version should match")
}
