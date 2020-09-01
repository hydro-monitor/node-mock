package trigger

import (
	"sync"
	"time"

	"github.com/golang/glog"
)

// Trigger represents a measurement trigger
type Trigger struct {
	timer         *time.Ticker
	interval      time.Duration // In seconds
	measurer_chan chan int
	analyzer_chan chan int
	stop_chan     chan int
	wg            *sync.WaitGroup
}

// NewTrigger creates and returns a new measurement trigger
func NewTrigger(interval int, measurer_chan chan int, analyzer_chan chan int, wg *sync.WaitGroup) *Trigger {
	return &Trigger{
		interval:      time.Duration(interval) * time.Second,
		measurer_chan: measurer_chan,
		analyzer_chan: analyzer_chan,
		stop_chan:     make(chan int),
		wg:            wg,
	}
}

// Start starts measurement trigger process. Exits when stop is received
func (t *Trigger) Start() error {
	t.timer = time.NewTicker(t.interval)
	for {
		select {
		case newInterval := <-t.analyzer_chan:
			newIntervalDuration := time.Duration(newInterval) * time.Second
			glog.Infof("Interval received is %v, while current interval is: %v", newIntervalDuration, t.interval)
			if (newIntervalDuration != t.interval) {
				glog.Infof("Interval received is %v, differs from current interval: %v. Updating ticker", newIntervalDuration, t.interval)			
				t.interval = newIntervalDuration
				t.timer.Stop()
				t.timer = time.NewTicker(t.interval)
				glog.Infof("Old timer stopped. New interval is: %v", newIntervalDuration)
			} else {
				glog.Infof("Timer continues unchanged")
			}
		case time := <-t.timer.C:
			glog.Infof("Tick at %v. Awaking Measurer", time)
			t.measurer_chan <- 1
		case <-t.stop_chan:
			glog.Info("Received stop sign")
			return nil
		}
	}
}

// Stop stops measurement trigger process
func (t *Trigger) Stop() error {
	t.timer.Stop()
	glog.Info("Timer stopped")
	glog.Info("Sending stop sign")
	t.stop_chan <- 1
	defer t.wg.Done()
	return nil
}
