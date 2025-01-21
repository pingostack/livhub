package plugo

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type testConfig struct {
	Name    string
	Version int
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
	onChange := func(ctx context.Context, cfg interface{}) error {
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
