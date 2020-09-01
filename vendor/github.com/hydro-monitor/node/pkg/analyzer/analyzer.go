package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/hydro-monitor/node/pkg/config"
	"github.com/hydro-monitor/node/pkg/envconfig"
)

// Analyzer represents a measurement analyzer
type Analyzer struct {
	wg                    *sync.WaitGroup
	trigger_chan          chan int
	measurer_chan         chan float64
	config_watcher_chan   chan *config.Configutation
	stop_chan             chan int
	config                *config.Configutation
	currentState          string
	intervalUpdateTimeout time.Duration
}

// updateConfiguration saves new node configuration
func (a *Analyzer) updateConfiguration(newConfig *config.Configutation) error {
	glog.Info("Saving node configuration")
	a.config = newConfig
	return nil
}

// NewAnalyzer creates and returns a new analyzer
func NewAnalyzer(measurer_chan chan float64, trigger_chan chan int, config_watcher_chan chan *config.Configutation, wg *sync.WaitGroup) *Analyzer {
	env := envconfig.New()
	a := &Analyzer{
		wg:                    wg,
		trigger_chan:          trigger_chan,
		measurer_chan:         measurer_chan,
		config_watcher_chan:   config_watcher_chan,
		stop_chan:             make(chan int),
		intervalUpdateTimeout: time.Duration(env.IntervalUpdateTimeout) * time.Second,
	}
	return a
}

// lookForCurrentState receives a measurement and checks limits from configuration states in order to identify current node state.
// Returns current node state name.
func (a *Analyzer) lookForCurrentState(measurement float64) (string, error) {
	for _, stateName := range a.config.GetStates() {
		// By design, if measurement is equal to lower limit, it is covered by the state
		if measurement >= a.config.GetState(stateName).LowerLimit && measurement < a.config.GetState(stateName).UpperLimit {
			return stateName, nil
		}
	}

	// If default state, that is the current state
	if a.config.HasDefaultState() {
		return a.config.GetDefaultStateName(), nil
	}

	glog.Errorf("Could not found current state for measurement %f", measurement)
	return "", fmt.Errorf("Could not found current state for measurement %f", measurement)
}

// updateCurrentState saves new state name and sends new interval to measurement trigger process
func (a *Analyzer) updateCurrentState(newStateName string) {
	glog.Infof("Current state is %s", newStateName)
	a.currentState = newStateName
	newInterval := a.config.GetState(newStateName).Interval
	glog.Infof("Sending new current interval (%d) to Trigger", newInterval)
	select {
	case a.trigger_chan <- newInterval:
		glog.Info("Interval update sent")
	case <-time.After(a.intervalUpdateTimeout):
		glog.Warning("Interval update timed out")
	}
}

// lookForAndUpdateState receives a measurement, looks for current state and issues an status update.
// If current state is not found, state stays as is. 
func (a *Analyzer) lookForAndUpdateState(measurement float64) {
	newState, err := a.lookForCurrentState(measurement)
	if err != nil {
		glog.Errorf("Could not found next state, staying at current state %s. Error: %v", a.currentState, err)
	} else {
		a.updateCurrentState(newState)
	}
}

// analyze receives a measurement and checks if it is still within the limits of the current state. 
// If not, it issues a state update.
// If current state is not set, it looks for it in the current node configuration. 
// If no current state is found analysis is skipped.
func (a *Analyzer) analyze(measurement float64) {
	glog.Info("Analyzing measurement")
	if a.currentState == "" {
		glog.Info("Current state unset. Setting current state")
		if currentStateName, err := a.lookForCurrentState(measurement); err != nil {
			glog.Info("Current state not found, skipping analysis")
			return
		} else {
			a.updateCurrentState(currentStateName)
			glog.Infof("Current state (%s) set successfully. Measurement analysis done", currentStateName)
			return
		}
	}

	if a.currentState == a.config.GetDefaultStateName() {
		if currentStateName, err := a.lookForCurrentState(measurement); err != nil {
			glog.Errorf("Could not found next state, staying at default state %s. Error: %v", a.currentState, err)
			// send current interval anyway in case there was a config update between measurements 
			a.updateCurrentState(a.config.GetDefaultStateName())
			return
		} else {
			if currentStateName == a.config.GetDefaultStateName() {
				glog.Infof("No limits were surpassed. Current state is (still) %s", a.currentState)
				// send current interval anyway in case there was a config update between measurements 
				a.updateCurrentState(a.config.GetDefaultStateName())
				return
			} else {
				a.updateCurrentState(currentStateName)
				return
			}
		}
	}

	// By design, if measurement is equal to lower limit, it is covered by the state 
	if measurement >= a.config.GetState(a.currentState).UpperLimit {
		glog.Info("Upper limit surpassed")
		a.lookForAndUpdateState(measurement)
	} else if measurement < a.config.GetState(a.currentState).LowerLimit {
		glog.Info("Lower limit surpassed")
		a.lookForAndUpdateState(measurement)
	} else {
		glog.Infof("No limits were surpassed. Current state is (still) %s", a.currentState)
		// send current interval anyway in case there was a config update between measurements 
		a.updateCurrentState(a.currentState)
	}
}

// Start starts analyzer process. Exits when stop is received
func (a *Analyzer) Start() error {
	defer a.wg.Done()
	for {
		select {
		case configuration := <-a.config_watcher_chan:
			glog.Infof("Configuration received: %v", configuration)
			a.updateConfiguration(configuration)
		case measurement := <-a.measurer_chan:
			glog.Infof("Measurement received: %f", measurement)
			if a.config == nil {
				glog.Info("Node configuration not loaded, skipping analysis")
				continue
			}
			a.analyze(measurement)
		case <-a.stop_chan:
			glog.Info("Received stop sign")
			return nil
		}
	}
}

// Stop stops analyzer process
func (a *Analyzer) Stop() error {
	glog.Info("Sending stop sign")
	a.stop_chan <- 1
	return nil
}
