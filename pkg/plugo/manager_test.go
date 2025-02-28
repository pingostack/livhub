package plugo

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

type testConfig struct {
	Name    string
	Version int
}

// Define test feature types
type TestFeatureA struct {
	Value string
}

func (f TestFeatureA) Type() interface{} {
	return TestFeatureA{}
}

type TestFeatureB struct {
	Value int
}

func (f TestFeatureB) Type() interface{} {
	return TestFeatureB{}
}

// Define an interface and a struct implementing it for interface matching tests
type FeatureInterface interface {
	GetName() string
}

type FeatureImpl struct {
	Name string
}

func (f FeatureImpl) GetName() string {
	return f.Name
}

func (f FeatureImpl) Type() interface{} {
	return FeatureImpl{}
}

// Embedded structure needed for callback testing
type callbackResult struct {
	called   bool
	mu       sync.Mutex
	features []interface{}
}

func (r *callbackResult) markCalled(features ...interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.called = true
	r.features = features
}

func (r *callbackResult) wasCalled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.called
}

func TestWithConfig(t *testing.T) {
	// Create temporary config file
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configFile, []byte(`
test.config:
  version: 2
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test default config
	defaultCfg := &testConfig{
		Name:    "test",
		Version: 1,
	}

	var receivedCfg interface{}
	// Create a test onChange function to capture config
	onChange := func(ctx context.Context, obj Plugin, cfg interface{}) error {
		receivedCfg = cfg
		return nil
	}

	// Register plugin
	err = RegisterPlugin("test-plugin", WithConfig("test.config", defaultCfg, onChange))
	if err != nil {
		t.Fatal(err)
	}
	defer UnregisterPlugin("test-plugin")

	// Set config provider
	err = SetConfigProvider(&FileProvider{
		path:   configFile,
		format: "yaml",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait a short time to ensure config is loaded
	time.Sleep(100 * time.Millisecond)

	// Verify received config
	received, ok := receivedCfg.(*testConfig)
	if !ok {
		t.Fatalf("expected receivedCfg to be *testConfig, got %T", receivedCfg)
	}

	expectedCfg := &testConfig{
		Name:    "test",
		Version: 2,
	}

	if !reflect.DeepEqual(received, expectedCfg) {
		t.Errorf("expected received config to be %+v, got %+v", expectedCfg, received)
	}
}

func TestBasicFeatureRequirements(t *testing.T) {
	// Create a new manager to avoid using global state
	mgr := newManager(context.Background())

	// Test result
	result := &callbackResult{}

	// Register callback requiring two features
	err := mgr.requireFeatures(func(a TestFeatureA, b TestFeatureB) {
		result.markCalled(a, b)
	})

	if err != nil {
		t.Fatalf("requireFeatures returned error: %v", err)
	}

	// Add first feature, callback should not trigger
	mgr.addFeature(TestFeatureA{Value: "test"})

	if result.wasCalled() {
		t.Fatal("Callback was triggered too early")
	}

	// Add second feature, callback should trigger
	mgr.addFeature(TestFeatureB{Value: 42})

	// Give callback some time to execute
	time.Sleep(50 * time.Millisecond)

	if !result.wasCalled() {
		t.Fatal("Callback was not triggered after all features were added")
	}
}

func TestInterfaceMatching(t *testing.T) {
	// Create a new manager
	mgr := newManager(context.Background())

	// Test result
	result := &callbackResult{}

	// Register callback requiring interface
	err := mgr.requireFeatures(func(f FeatureInterface) {
		result.markCalled(f)
	})

	if err != nil {
		t.Fatalf("requireFeatures returned error: %v", err)
	}

	// Add feature implementing the interface
	mgr.addFeature(FeatureImpl{Name: "test-interface"})

	// Give callback some time to execute
	time.Sleep(50 * time.Millisecond)

	if !result.wasCalled() {
		t.Fatal("Callback was not triggered after interface implementation was added")
	}
}

func TestNestedFeatureAddition(t *testing.T) {
	// Create a new manager
	mgr := newManager(context.Background())

	// Test multi-level callbacks
	firstResult := &callbackResult{}
	secondResult := &callbackResult{}

	// Callback 1: Requires feature A, adds feature C
	err := mgr.requireFeatures(func(a TestFeatureA) {
		firstResult.markCalled(a)
		// Add new feature in callback
		mgr.addFeature(TestFeatureC{Value: "from-callback"})
	})

	if err != nil {
		t.Fatalf("First requireFeatures returned error: %v", err)
	}

	// Callback 2: Requires feature C
	err = mgr.requireFeatures(func(c TestFeatureC) {
		secondResult.markCalled(c)
	})

	if err != nil {
		t.Fatalf("Second requireFeatures returned error: %v", err)
	}

	// Add feature that triggers first callback
	mgr.addFeature(TestFeatureA{Value: "test"})

	// Give callback chain some time to execute
	time.Sleep(100 * time.Millisecond)

	// Verify both callbacks were triggered
	if !firstResult.wasCalled() {
		t.Fatal("First callback was not triggered")
	}

	if !secondResult.wasCalled() {
		t.Fatal("Second callback was not triggered")
	}
}

func TestCallbackTriggeredOnlyOnce(t *testing.T) {
	// Create a new manager
	mgr := newManager(context.Background())

	// Counter
	counter := 0
	var mu sync.Mutex

	// Register callback
	err := mgr.requireFeatures(func(a TestFeatureA) {
		mu.Lock()
		counter++
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("requireFeatures returned error: %v", err)
	}

	// Add feature
	mgr.addFeature(TestFeatureA{Value: "test"})

	// Add same feature again
	mgr.addFeature(TestFeatureA{Value: "another"})

	// Give callback some time to execute
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := counter
	mu.Unlock()

	if count != 1 {
		t.Fatalf("Callback was triggered %d times, expected 1", count)
	}
}

// Additional type defined for nested feature tests
type TestFeatureC struct {
	Value string
}

func (f TestFeatureC) Type() interface{} {
	return TestFeatureC{}
}
