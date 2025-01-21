package plugo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/pingostack/livhub/pkg/logger"
	"github.com/spf13/viper"
)

// ConfigProvider defines the interface for configuration providers
type ConfigProvider interface {
	// Read returns the configuration content and format
	Read(ctx context.Context) (io.Reader, string, error)
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

// readConfig reads configuration from the provider into viper and notifies manager
func (l *ConfigLoader) readConfig() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	reader, format, err := l.provider.Read(l.ctx)
	if err != nil {
		return fmt.Errorf("failed to read from provider: %w", err)
	}
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			closer.Close()
		}
	}()

	// Create a buffer to store the content
	var buf bytes.Buffer
	teeReader := io.TeeReader(reader, &buf)

	if format != "" {
		l.viper.SetConfigType(format)
	}

	// Read config using the tee reader
	if err := l.viper.ReadConfig(teeReader); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Notify about the update with latest viper content
	if l.onUpdate != nil {
		if err := l.onUpdate(l.viper); err != nil {
			return fmt.Errorf("failed to handle config update: %w", err)
		}
	}

	return nil
}

// SetProvider sets the configuration provider
func (l *ConfigLoader) SetProvider(provider ConfigProvider) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.provider = provider

	// Get the initial content
	reader, format, err := provider.Read(l.ctx)
	if err != nil {
		return fmt.Errorf("failed to read from provider: %w", err)
	}
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			closer.Close()
		}
	}()

	// Create a buffer to store the content
	var buf bytes.Buffer
	teeReader := io.TeeReader(reader, &buf)

	if format != "" {
		l.viper.SetConfigType(format)
	}

	// Read config using the tee reader
	if err := l.viper.ReadConfig(teeReader); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Set up file watching if it's a file provider

	l.viper.SetConfigFile(provider.LocalPath())
	l.viper.OnConfigChange(func(in fsnotify.Event) {
		if err := l.readConfig(); err != nil {
			logger.WithError(err).Warnf("Failed to read config")
		}
	})
	l.viper.WatchConfig()

	// Notify about the update with latest viper content
	if l.onUpdate != nil {
		if err := l.onUpdate(l.viper); err != nil {
			return fmt.Errorf("failed to handle config update: %w", err)
		}
	}

	return nil
}

// LoadConfig loads the configuration for the registered type
func (l *ConfigLoader) LoadConfig() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.provider == nil {
		return fmt.Errorf("no provider set")
	}

	return l.readConfig()
}

// GetViper returns the viper instance
func (l *ConfigLoader) GetViper() *viper.Viper {
	return l.viper
}

// Stop stops the configuration loader and watcher
func (l *ConfigLoader) Stop() {
	l.cancel()
}
