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

func WithCreate(create func() (interface{}, error)) registerOption {
	return func(m *manager, pi *plugInfo) {
		obj, err := create()
		if err != nil {
			panic(err)
		}
		pi.obj = obj
	}
}

func WithInit(init func(ctx context.Context, pl Plugin) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.init = init
	}
}

func WithPre(pre func(ctx context.Context, pl Plugin) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.pre = pre
	}
}

func WithRun(run func(ctx context.Context, pl Plugin) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.run = run
	}
}

func WithExit(exit func(ctx context.Context, pl Plugin) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.exit = exit
	}
}

func WithConfig(key string, defaultCfg interface{}, onChange func(ctx context.Context, pl Plugin, cfg interface{}) error) registerOption {
	return func(m *manager, pi *plugInfo) {
		pi.config = configInfo{
			configKey:    key,
			configType:   reflect.TypeOf(defaultCfg),
			defaultValue: defaultCfg,
		}
		pi.onConfigChange = onChange
	}
}

type Feature interface{}

type manager struct {
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	plugins      []*plugInfo
	wg           sync.WaitGroup
	configLoader *ConfigLoader
	closeOnce    sync.Once
	callbacks    []callbackInfo
	features     map[reflect.Type]Feature
}

type callbackInfo struct {
	fn         reflect.Value
	paramTypes []reflect.Type
	triggered  bool
}

func newManager(ctx context.Context) *manager {
	ctx, cancel := context.WithCancel(ctx)
	return &manager{
		ctx:       ctx,
		cancel:    cancel,
		plugins:   make([]*plugInfo, 0),
		callbacks: make([]callbackInfo, 0),
		features:  make(map[reflect.Type]Feature),
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
			if err := p.init(m.ctx, p.obj); err != nil {
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
			if err := p.pre(m.ctx, p.obj); err != nil {
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
				chErr <- p.run(m.ctx, p.obj)
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
			p.exit(m.ctx, p.obj)
		}
	}
	return nil
}

func (m *manager) requireFeatures(callback ...interface{}) error {
	for _, cb := range callback {
		fnValue := reflect.ValueOf(cb)
		if fnValue.Kind() != reflect.Func {
			return fmt.Errorf("expected function, got %T", cb)
		}

		fnType := fnValue.Type()
		numParams := fnType.NumIn()
		paramTypes := make([]reflect.Type, numParams)

		for i := 0; i < numParams; i++ {
			paramTypes[i] = fnType.In(i)
		}

		m.callbacks = append(m.callbacks, callbackInfo{
			fn:         fnValue,
			paramTypes: paramTypes,
			triggered:  false,
		})

		m.tryTriggerCallback(len(m.callbacks) - 1)
	}

	return nil
}

func (m *manager) addFeature(feature Feature) {
	featureType := reflect.TypeOf(feature)
	m.features[featureType] = feature

	for i := range m.callbacks {
		if !m.callbacks[i].triggered {
			m.tryTriggerCallback(i)
		}
	}
}

func (m *manager) tryTriggerCallback(index int) {
	if index >= len(m.callbacks) || m.callbacks[index].triggered {
		return
	}

	callback := m.callbacks[index]

	params := make([]reflect.Value, len(callback.paramTypes))
	allAvailable := true

	for i, paramType := range callback.paramTypes {
		feature, ok := m.features[paramType]
		if !ok {
			if paramType.Kind() == reflect.Interface {
				found := false
				for ft, f := range m.features {
					if ft.Implements(paramType) {
						params[i] = reflect.ValueOf(f)
						found = true
						break
					}
				}
				if !found {
					allAvailable = false
					break
				}
			} else {
				allAvailable = false
				break
			}
		} else {
			params[i] = reflect.ValueOf(feature)
		}
	}

	if allAvailable {
		m.callbacks[index].triggered = true
		callback.fn.Call(params)
	}
}

func (m *manager) handleConfigUpdate(v *viper.Viper) error {
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

		// Call the onChange handler
		if err := p.onConfigChange(m.ctx, p.obj, newConfig); err != nil {
			return fmt.Errorf("failed to handle config change for plugin %s: %w", p.name, err)
		}

		// Store the new config as last config after successful update
		p.config.lastConfig = newConfig
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

func RequireFeatures(ctx context.Context, callback ...interface{}) error {
	return defaultManager.requireFeatures(callback...)
}

func AddFeature(feature Feature) {
	defaultManager.addFeature(feature)
}
