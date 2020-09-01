package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/golang/glog"
	"github.com/joho/godotenv"

	"github.com/hydro-monitor/node-mock/pkg/measurer"
	"github.com/hydro-monitor/node/pkg/analyzer"
	"github.com/hydro-monitor/node/pkg/config"
	"github.com/hydro-monitor/node/pkg/envconfig"
	"github.com/hydro-monitor/node/pkg/manual"
	"github.com/hydro-monitor/node/pkg/photocleaner"
	"github.com/hydro-monitor/node/pkg/trigger"
)

func init() {
	flag.Set("logtostderr", "true")

	// Loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		glog.Infof("No .env file found")
	}
}

// node represents a hydro monitor node with all it's correspondant processes
type node struct {
	t  *trigger.Trigger
	m  *measurer.Measurer
	a  *analyzer.Analyzer
	cw *config.ConfigWatcher
	mt *manual.ManualMeasurementTrigger
	pc *photocleaner.PhotoCleaner
}

// NewNode creates and returns a new node with all it's correspondant processes
func newNode(triggerMeasurer, triggerAnalyzer, triggerConfig, manualMeasurer chan int, measurerAnalyzer chan float64, configAnalyzer chan *config.Configutation, wg *sync.WaitGroup) *node {
	env := envconfig.New()
	glog.Infof("Env config: %v", env)

	return &node{
		t:  trigger.NewTrigger(env.InitialTriggerInterval, triggerMeasurer, triggerAnalyzer, wg),
		m:  measurer.NewMeasurer(triggerMeasurer, manualMeasurer, measurerAnalyzer, wg),
		a:  analyzer.NewAnalyzer(measurerAnalyzer, triggerAnalyzer, configAnalyzer, wg),
		cw: config.NewConfigWatcher(triggerConfig, configAnalyzer, env.ConfigurationUpdateInterval, wg),
		mt: manual.NewManualMeasurementTrigger(manualMeasurer, env.ManualMeasurementPollInterval, wg),
		pc: photocleaner.NewPhotoCleaner(env.PhotoCleaningInterval, wg),
	}
}

// main function creates all the channels needed for inter process communication, a waitgroup 
// hat matches the amount of processes, and a new node.
// Node's processes are started and a channel is created to wait for SIGINT and SIGTERM signals. 
// If any of these signals are received, all node processes are commanded to stop.
// main waits for all node processes to exit gracefully and then returns.
func main() {
	flag.Parse()
	var wg sync.WaitGroup
	wg.Add(6)
	triggerMeasurer := make(chan int)
	triggerAnalyzer := make(chan int)
	measurerAnalyzer := make(chan float64)
	triggerConfig := make(chan int)
	manualMeasurer := make(chan int)
	configAnalyzer := make(chan *config.Configutation)
	n := newNode(triggerMeasurer, triggerAnalyzer, triggerConfig, manualMeasurer, measurerAnalyzer, configAnalyzer, &wg)

	go n.a.Start()
	go n.m.Start()
	go n.t.Start()
	go n.cw.Start()
	go n.mt.Start()
	go n.pc.Start()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	glog.Info("Awaiting signal")
	sig := <-sigs
	glog.Infof("Signal received: %v. Stopping workers", sig)

	n.t.Stop()
	n.m.Stop()
	n.a.Stop()
	n.cw.Stop()
	n.mt.Stop()
	n.pc.Stop()

	wg.Wait()
}
