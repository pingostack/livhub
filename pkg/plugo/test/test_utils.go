package test

import (
	"encoding/json"
	"sync/atomic"
)

// TestPlugin represents a test plugin with lifecycle tracking
type TestPlugin struct {
	initCalled atomic.Bool
	preCalled  atomic.Bool
	runCalled  atomic.Bool
	exitCalled atomic.Bool
	config     *TestConfig
}

// TestConfig represents test configuration
type TestConfig struct {
	Name    string `json:"name" yaml:"name"`
	Version int    `json:"version" yaml:"version"`
}

func (c *TestConfig) String() string {
	jbuf, _ := json.Marshal(c)
	return string(jbuf)
}
