package test

import (
	"encoding/json"
	"sync/atomic"
)

// TestPlugin represents a test plugin with lifecycle tracking
type TestPlugin struct {
	setupCalled atomic.Bool
	runCalled   atomic.Bool
	exitCalled  atomic.Bool
	config      *TestConfig
}

func (p *TestPlugin) Type() interface{} {
	return &TestPlugin{}
}

// TestConfig represents test configuration
type TestConfig struct {
	Name    string `mapstructure:"name"`
	Version int    `mapstructure:"version"`
	NodeID  string `mapstructure:"node_id"`
}

func (c *TestConfig) String() string {
	jbuf, _ := json.Marshal(c)
	return string(jbuf)
}
