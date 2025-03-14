package plugo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var (
	ErrPluginAlreadyExists = errors.New("plugin already exists")
	ErrPluginNotFound      = errors.New("plugin not found")
	ErrConfigKeyNotFound   = errors.New("config key not found")
	ErrConfigTypeMismatch  = errors.New("config type mismatch")
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

type registerOption func(m *pluginManager, pi *plugInfo)

func WithCreate(create func(ctx context.Context) Plugin) registerOption {
	return func(m *pluginManager, pi *plugInfo) {
		pi.create = create
	}
}

func WithSetup(setup func(ctx context.Context, pl Plugin) error) registerOption {
	return func(m *pluginManager, pi *plugInfo) {
		pi.setup = setup
	}
}

func WithRun(run func(ctx context.Context, pl Plugin) error) registerOption {
	return func(m *pluginManager, pi *plugInfo) {
		pi.run = run
	}
}

func WithExit(exit func(ctx context.Context, pl Plugin) error) registerOption {
	return func(m *pluginManager, pi *plugInfo) {
		pi.exit = exit
	}
}

func WithConfig(key string, defaultCfg interface{}, onChange func(ctx context.Context, pl Plugin, cfg interface{}) error) registerOption {
	return func(m *pluginManager, pi *plugInfo) {
		pi.config = configInfo{
			configKey:    key,
			configType:   reflect.TypeOf(defaultCfg),
			defaultValue: defaultCfg,
		}
		pi.onConfigChange = onChange
	}
}

// WithCritical marks a plugin as critical
// When a critical plugin's run function abnormally exits, it will automatically trigger pluginManager shutdown
func WithCritical() registerOption {
	return func(m *pluginManager, pi *plugInfo) {
		pi.critical = true
	}
}

type pluginManager struct {
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	plugins      []*plugInfo
	wg           sync.WaitGroup
	configLoader *ConfigLoader
	closeOnce    sync.Once
}

func newManager(ctx context.Context) *pluginManager {
	ctx, cancel := context.WithCancel(ctx)
	m := &pluginManager{
		ctx:     ctx,
		cancel:  cancel,
		plugins: make([]*plugInfo, 0),
	}

	m.configLoader = NewConfigLoader(ctx, m.handleConfigUpdate)

	go func() {
		<-ctx.Done()
		m.exit()
	}()

	return m
}

func (m *pluginManager) register(name string, options ...registerOption) error {
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

func (m *pluginManager) unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.plugins {
		if p.name == name {
			m.plugins = append(m.plugins[:i], m.plugins[i+1:]...)
			return
		}
	}
}

func (m *pluginManager) setupPlugin(p *plugInfo) error {
	// Only call setup if not already set up
	if atomic.CompareAndSwapInt32(&p.setupComplete, 0, 1) && p.setup != nil {
		if err := p.setup(m.ctx, p.obj); err != nil {
			// Reset flag on failure
			atomic.StoreInt32(&p.setupComplete, 0)
			return fmt.Errorf("failed to setup plugin %s: %w", p.name, err)
		}
	}
	return nil
}

func (m *pluginManager) runPlugin(p *plugInfo) {
	// Only start running if set up and not already running
	if p.run != nil && p.setupComplete == 1 && atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		m.wg.Add(1)
		go func(p *plugInfo) {
			defer func() {
				// if r := recover(); r != nil {
				// 	log.Printf("Error running plugin %s: %v\n", p.name, r)
				// 	// Only trigger pluginManager shutdown if this is a critical plugin
				// 	if p.critical {
				// 		log.Printf("Critical plugin %s panicked, shutting down pluginManager", p.name)
				// 		go m.Close()
				// 	}
				// }
				m.wg.Done()
			}()

			if err := p.run(m.ctx, p.obj); err != nil {
				// Log the error
				log.Printf("Error running plugin %s: %v\n", p.name, err)
				// Only trigger pluginManager shutdown if this is a critical plugin
				if p.critical {
					log.Printf("Critical plugin %s failed, shutting down pluginManager", p.name)
					go m.Close()
				}
			}
		}(p)
	}
}

// Initialize a newly created plugin and move it through its lifecycle
func (m *pluginManager) startupPlugin(p *plugInfo) error {
	// First setup the plugin
	if err := m.setupPlugin(p); err != nil {
		return err
	}

	// Then run it
	m.runPlugin(p)

	return nil
}

func (m *pluginManager) exit() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.plugins {
		if p.exit != nil {
			p.exit(m.ctx, p.obj)
		}
	}
	return nil
}

func (m *pluginManager) handleConfigUpdate(v *viper.Viper) error {
	for _, p := range m.plugins {
		if p.onConfigChange == nil || p.config.configKey == "" {
			continue
		}

		newConfig := reflect.New(p.config.configType.Elem()).Interface()

		// Set default values first
		if p.config.defaultValue != nil {
			if err := mapstructure.Decode(p.config.defaultValue, newConfig); err != nil {
				return fmt.Errorf("failed to decode default config: %w", err)
			}
		}

		// Check if the key exists in the current configuration
		if !v.IsSet(p.config.configKey) {
			log.Printf("Configuration key %s not present, skipping\n", p.config.configKey)
			continue // Skip if configuration key is not present
		}

		// Unmarshal current config values, overriding defaults
		if err := v.UnmarshalKey(p.config.configKey, newConfig); err != nil {
			return fmt.Errorf("failed to unmarshal config for key %s: %w", p.config.configKey, err)
		}

		// Compare with last config if it exists
		if p.config.lastConfig != nil {
			// Check if configs are equal
			if reflect.DeepEqual(p.config.lastConfig, newConfig) {
				continue // Skip if no changes
			}
		}

		isFirstConfigLoad := atomic.CompareAndSwapInt32(&p.configLoaded, 0, 1)

		if p.create != nil && p.obj == nil {
			p.obj = p.create(m.ctx)
		}

		if err := p.onConfigChange(m.ctx, p.obj, newConfig); err != nil {
			return fmt.Errorf("failed to handle config change for plugin %s: %w", p.name, err)
		}

		if isFirstConfigLoad {
			if err := m.startupPlugin(p); err != nil {
				return err
			}
		}

		// Store the new config as last config after successful update
		p.config.lastConfig = newConfig
	}

	return nil
}

func (m *pluginManager) Start() error {
	return m.configLoader.Start()
}

func (m *pluginManager) Close() error {
	m.closeOnce.Do(func() {
		m.cancel()
	})
	return nil
}

func (m *pluginManager) Wait() {
	m.wg.Wait()
	<-m.ctx.Done()
}
