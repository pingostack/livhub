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

// 定义测试用的特性类型
type TestFeatureA struct {
	Value string
}

type TestFeatureB struct {
	Value int
}

// 定义一个接口和实现该接口的结构体，用于测试接口匹配
type FeatureInterface interface {
	GetName() string
}

type FeatureImpl struct {
	Name string
}

func (f FeatureImpl) GetName() string {
	return f.Name
}

// 测试回调需要的内嵌结构
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
	// 创建临时配置文件
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configFile, []byte(`
test.config:
  version: 2
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 创建一个测试用的默认配置
	defaultCfg := &testConfig{
		Name:    "test",
		Version: 1,
	}

	var receivedCfg interface{}
	// 创建一个测试用的onChange函数来捕获配置
	onChange := func(ctx context.Context, obj Plugin, cfg interface{}) error {
		receivedCfg = cfg
		return nil
	}

	// 注册插件
	err = RegisterPlugin("test-plugin", WithConfig("test.config", defaultCfg, onChange))
	if err != nil {
		t.Fatal(err)
	}
	defer UnregisterPlugin("test-plugin")

	// 设置配置提供者
	err = SetConfigProvider(&FileProvider{
		path:   configFile,
		format: "yaml",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 等待一小段时间确保配置加载完成
	time.Sleep(100 * time.Millisecond)

	// 验证接收到的配置
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
	// 创建一个新的管理器以避免使用全局状态
	mgr := newManager(context.Background())

	// 测试结果
	result := &callbackResult{}

	// 注册需要两个特性的回调
	err := mgr.requireFeatures(func(a TestFeatureA, b TestFeatureB) {
		result.markCalled(a, b)
	})

	if err != nil {
		t.Fatalf("requireFeatures returned error: %v", err)
	}

	// 添加第一个特性，回调不应该触发
	mgr.addFeature(TestFeatureA{Value: "test"})

	if result.wasCalled() {
		t.Fatal("Callback was triggered too early")
	}

	// 添加第二个特性，回调应该触发
	mgr.addFeature(TestFeatureB{Value: 42})

	// 给回调一些时间执行
	time.Sleep(50 * time.Millisecond)

	if !result.wasCalled() {
		t.Fatal("Callback was not triggered after all features were added")
	}
}

func TestInterfaceMatching(t *testing.T) {
	// 创建一个新的管理器
	mgr := newManager(context.Background())

	// 测试结果
	result := &callbackResult{}

	// 注册需要接口的回调
	err := mgr.requireFeatures(func(f FeatureInterface) {
		result.markCalled(f)
	})

	if err != nil {
		t.Fatalf("requireFeatures returned error: %v", err)
	}

	// 添加实现该接口的特性
	mgr.addFeature(FeatureImpl{Name: "test-interface"})

	// 给回调一些时间执行
	time.Sleep(50 * time.Millisecond)

	if !result.wasCalled() {
		t.Fatal("Callback was not triggered after interface implementation was added")
	}
}

func TestNestedFeatureAddition(t *testing.T) {
	// 创建一个新的管理器
	mgr := newManager(context.Background())

	// 测试多级回调
	firstResult := &callbackResult{}
	secondResult := &callbackResult{}

	// 回调1: 需要特性A，添加特性C
	err := mgr.requireFeatures(func(a TestFeatureA) {
		firstResult.markCalled(a)
		// 在回调中添加新特性
		mgr.addFeature(TestFeatureC{Value: "from-callback"})
	})

	if err != nil {
		t.Fatalf("First requireFeatures returned error: %v", err)
	}

	// 回调2: 需要特性C
	err = mgr.requireFeatures(func(c TestFeatureC) {
		secondResult.markCalled(c)
	})

	if err != nil {
		t.Fatalf("Second requireFeatures returned error: %v", err)
	}

	// 添加触发第一个回调的特性
	mgr.addFeature(TestFeatureA{Value: "test"})

	// 给回调链一些时间执行
	time.Sleep(100 * time.Millisecond)

	// 验证两个回调是否都被触发
	if !firstResult.wasCalled() {
		t.Fatal("First callback was not triggered")
	}

	if !secondResult.wasCalled() {
		t.Fatal("Second callback was not triggered")
	}
}

func TestCallbackTriggeredOnlyOnce(t *testing.T) {
	// 创建一个新的管理器
	mgr := newManager(context.Background())

	// 计数器
	counter := 0
	var mu sync.Mutex

	// 注册回调
	err := mgr.requireFeatures(func(a TestFeatureA) {
		mu.Lock()
		counter++
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("requireFeatures returned error: %v", err)
	}

	// 添加特性
	mgr.addFeature(TestFeatureA{Value: "test"})

	// 再次添加相同特性
	mgr.addFeature(TestFeatureA{Value: "another"})

	// 给回调一些时间执行
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := counter
	mu.Unlock()

	if count != 1 {
		t.Fatalf("Callback was triggered %d times, expected 1", count)
	}
}

// 为嵌套特性测试定义的额外类型
type TestFeatureC struct {
	Value string
}
