package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/hydro-monitor/node/pkg/envconfig"
	"github.com/hydro-monitor/node/pkg/server"
)

const (
	defaultStateName = "default"
)

// Configutation is a map with all posible states in the node configuration
type Configutation struct {
	stateNames   []string
	states       map[string]server.State
}

// NewConfiguration creates and returns a configuration with all its states
func NewConfiguration(states map[string]server.State) *Configutation {
	stateNames := []string{}
	for k := range states {
		if k == defaultStateName {
			continue
		}
		stateNames = append(stateNames, k)
	}

	return &Configutation{
		stateNames:   stateNames,
		states:       states,
	}
}

// GetStates returns an array with the state names
func (c *Configutation) GetStates() []string {
	return c.stateNames
}

// GetState receives a state name, returns the state struct
func (c *Configutation) GetState(stateName string) server.State {
	return c.states[stateName]
}

// GetDefaultState returns the default state name string
func (c *Configutation) GetDefaultStateName() string {
	return defaultStateName
}

// HasDefaultState returns true if configuration has a default state
func (c *Configutation) HasDefaultState() bool {
	_, ok := c.states[defaultStateName]
	return ok
}

// ConfigWatcher continuously polls the servers for the right configuration of the node
type ConfigWatcher struct {
	wg                         *sync.WaitGroup
	trigger_chan               chan int
	analyzer_chan              chan *Configutation
	stop_chan                  chan int
	timer                      *time.Ticker
	interval                   time.Duration // In seconds
	server                     *server.Server
	configurationUpdateTimeout time.Duration
}

// NewConfigWatcher creates and returns a new config watcher
func NewConfigWatcher(trigger_chan chan int, analyzer_chan chan *Configutation, interval int, wg *sync.WaitGroup) *ConfigWatcher {
	env := envconfig.New()
	c := &ConfigWatcher{
		wg:                         wg,
		trigger_chan:               trigger_chan,
		analyzer_chan:              analyzer_chan,
		stop_chan:                  make(chan int),
		interval:                   time.Duration(interval),
		server:                     server.NewServer(),
		configurationUpdateTimeout: time.Duration(env.ConfigurationUpdateTimeout) * time.Second,
	}
	return c
}

// updateConfiguration gets node configuration from hydro monitor server.
// Sends configuration update to analyzer process
func (c *ConfigWatcher) updateConfiguration() error {
	serverConfig, err := c.server.GetNodeConfiguration()
	if err != nil {
		glog.Errorf("Could not get configuration from server: %v", err)
		return err
	}
	config := NewConfiguration(serverConfig.States)
	glog.Infof("Sending new node configuration: %v", config)
	select {
	case c.analyzer_chan <- config:
		glog.Info("Current configuration sent")
		return nil
	case <-time.After(c.configurationUpdateTimeout):
		glog.Warning("Configuration send timed out")
		return fmt.Errorf("Configuration send timed out")
	}
}

// Start starts config watcher process. Exits when stop is received
func (c *ConfigWatcher) Start() error {
	defer c.wg.Done()
	glog.Infof("Quering server for node configuration.")
	c.updateConfiguration()
	c.timer = time.NewTicker(c.interval * time.Second)
	for {
		select {
		case time := <-c.timer.C:
			glog.Infof("Tick at %v. Quering server for node configuration.", time)
			c.updateConfiguration()
		case <-c.stop_chan:
			glog.Info("Received stop from sign")
			return nil
		}
	}
}

// Stop stops config watcher process
func (c *ConfigWatcher) Stop() error {
	c.timer.Stop()
	glog.Info("Timer stopped")
	glog.Info("Sending stop sign")
	c.stop_chan <- 1
	return nil
}
