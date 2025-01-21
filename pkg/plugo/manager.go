package plugo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var (
	ErrPluginAlreadyExists = errors.New("plugin already exists")
	ErrPluginNotFound      = errors.New("plugin not found")
	ErrConfigKeyNotFound   = errors.New("config key not found")
	ErrConfigTypeMismatch  = errors.New("config type mismatch")
)

type registerOption func(m *manager, pi *plugInfo)

func WithFeatureType(obj interface{}) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.featureType = reflect.TypeOf(obj)
		pi.obj = obj
	}
}

func WithInit(init func(ctx context.Context) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.init = init
	}
}

func WithPre(pre func(ctx context.Context) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.pre = pre
	}
}

func WithRun(run func(ctx context.Context) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.run = run
	}
}

func WithExit(exit func(ctx context.Context) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.exit = exit
	}
}

func WithConfig(key string, defaultCfg interface{}, onChange func(ctx context.Context, cfg interface{}) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.config = configInfo{
			configKey:    key,
			configType:   reflect.TypeOf(defaultCfg),
			defaultValue: defaultCfg,
		}
		pi.onConfigChange = onChange
	}
}

type manager struct {
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	plugins      []*plugInfo
	wg           sync.WaitGroup
	configLoader *ConfigLoader
	closeOnce    sync.Once
}

func newManager(ctx context.Context) *manager {
	ctx, cancel := context.WithCancel(ctx)
	return &manager{
		ctx:     ctx,
		cancel:  cancel,
		plugins: make([]*plugInfo, 0),
	}
}

func (m *manager) register(name string, options ...registerOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.plugins {
		if p.name == name {
			return ErrPluginAlreadyExists
		}
	}

	pi := &plugInfo{
		name: name,
	}
	for _, option := range options {
		option(m, pi)
	}
	m.plugins = append(m.plugins, pi)
	return nil
}

func (m *manager) unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.plugins {
		if p.name == name {
			m.plugins = append(m.plugins[:i], m.plugins[i+1:]...)
			return
		}
	}
}

func (m *manager) init() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.plugins {
		if p.init != nil {
			if err := p.init(m.ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *manager) pre() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.plugins {
		if p.pre != nil {
			if err := p.pre(m.ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *manager) run() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chErr := make(chan error, len(m.plugins))

	for _, p := range m.plugins {
		if p.run != nil {
			m.wg.Add(1)
			go func(p *plugInfo) {
				defer m.wg.Done()
				chErr <- p.run(m.ctx)
			}(p)
		}
	}

	for i := 0; i < len(m.plugins); i++ {
		if err := <-chErr; err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) exit() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.plugins {
		if p.exit != nil {
			p.exit(m.ctx)
		}
	}
	return nil
}

func (m *manager) handleConfigUpdate(v *viper.Viper) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.plugins {
		if p.onConfigChange == nil {
			continue
		}

		// Create a new instance of the config type
		newConfig := reflect.New(p.config.configType.Elem()).Interface()

		// Set default values first
		if p.config.defaultValue != nil {
			if err := mapstructure.Decode(p.config.defaultValue, newConfig); err != nil {
				return fmt.Errorf("failed to decode default config: %w", err)
			}
		}

		// Unmarshal current config values, overriding defaults
		if err := v.UnmarshalKey(p.config.configKey, newConfig); err != nil {
			return fmt.Errorf("failed to unmarshal config for key %s: %w", p.config.configKey, err)
		}

		// Call the onChange handler
		if err := p.onConfigChange(m.ctx, newConfig); err != nil {
			return fmt.Errorf("failed to handle config change for plugin %s: %w", p.name, err)
		}
	}

	return nil
}

func (m *manager) close() error {
	m.closeOnce.Do(func() {
		m.cancel()
		m.wg.Wait()
		if err := m.exit(); err != nil {
			return
		}
	})
	return nil
}

var defaultManager *manager

func init() {
	defaultManager = newManager(context.Background())
	defaultManager.configLoader = NewConfigLoader(defaultManager.ctx, defaultManager.handleConfigUpdate)
}

func RegisterPlugin(name string, options ...registerOption) error {
	return defaultManager.register(name, options...)
}

func UnregisterPlugin(name string) {
	defaultManager.unregister(name)
}

func Start() error {
	err := defaultManager.init()
	if err != nil {
		return err
	}

	if err := defaultManager.pre(); err != nil {
		return err
	}

	go func() {
		var err error
		defer func() {
			if err != nil {
				defaultManager.close()
			}
		}()

		if err = defaultManager.run(); err != nil {
			return
		}
	}()

	return nil
}

func Close() error {
	return defaultManager.close()
}

func SetConfigProvider(provider ConfigProvider) error {
	return defaultManager.configLoader.SetProvider(provider)
}
