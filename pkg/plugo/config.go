package plugo

import (
	"context"
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ConfigProvider defines the interface for configuration providers
type ConfigProvider interface {
	// Read returns the configuration content and format
	// Read(ctx context.Context) (io.Reader, string, error)
	// String returns a string representation of the provider
	String() string
	// LocalPath returns the local path of the provider
	LocalPath() string
}

// OnConfigUpdate is called when configuration content is updated
type OnConfigUpdate func(v *viper.Viper) error

// ConfigLoader manages configuration loading and watching
type ConfigLoader struct {
	mu       sync.RWMutex
	provider ConfigProvider
	viper    *viper.Viper
	ctx      context.Context
	cancel   context.CancelFunc
	onUpdate OnConfigUpdate
}

// NewConfigLoader creates a new configuration loader instance
func NewConfigLoader(ctx context.Context, onUpdate OnConfigUpdate) *ConfigLoader {
	ctx, cancel := context.WithCancel(ctx)
	v := viper.New()
	v.SetConfigType("yaml") // Set default config type
	return &ConfigLoader{
		viper:    v,
		ctx:      ctx,
		cancel:   cancel,
		onUpdate: onUpdate,
	}
}

// SetProvider sets the configuration provider
func (l *ConfigLoader) SetProvider(provider ConfigProvider) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.provider = provider

	// Set up file watching
	l.viper.SetConfigFile(provider.LocalPath())
	l.viper.WatchConfig()
	l.viper.OnConfigChange(func(in fsnotify.Event) {
		if l.onUpdate != nil {
			l.onUpdate(l.viper)
		}
	})

	// Do initial read
	if err := l.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Notify about initial content
	if l.onUpdate != nil {
		l.onUpdate(l.viper)
	}

	return nil
}

// GetViper returns the viper instance
func (l *ConfigLoader) GetViper() *viper.Viper {
	return l.viper
}

// Stop stops the configuration loader and watcher
func (l *ConfigLoader) Stop() {
	l.cancel()
}
