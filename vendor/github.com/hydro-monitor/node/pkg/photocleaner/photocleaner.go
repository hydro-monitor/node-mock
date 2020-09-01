package photocleaner

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/hydro-monitor/node/pkg/envconfig"
)

// PhotoCleaner represents a photo cleaner
type PhotoCleaner struct {
	timer       *time.Ticker
	interval    time.Duration // In hours
	picturesDir string
	stop_chan   chan int
	wg          *sync.WaitGroup
}

// NewPhotoCleaner creates and returns new photo cleaner
func NewPhotoCleaner(interval int, wg *sync.WaitGroup) *PhotoCleaner {
	env := envconfig.New()
	return &PhotoCleaner{
		interval:    time.Duration(interval) * time.Hour,
		picturesDir: env.PicturesDir,
		stop_chan:   make(chan int),
		wg:          wg,
	}
}

// inTimeSpan returns true if check time is after start and before end time
func (pc *PhotoCleaner) inTimeSpan(start, end, check time.Time) bool {
    return check.After(start) && check.Before(end)
}

// photoNameToTime turns photo name to photo creation timestamp
func (pc *PhotoCleaner) photoNameToTime(photoName string) (*time.Time, error) {
	parts := strings.Split(photoName, " ")
	timeStr := strings.Join(parts[0:len(parts)-1], " ")
	// Format string that time.String() uses according to 
	// https://golang.org/pkg/time/#Time.String is 
	// "2006-01-02 15:04:05.999999999 -0700 MST"
	t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", timeStr)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// sweepPicturesDir iterates through files in picturesDir. Deletes pictures older than a week
func (pc *PhotoCleaner) sweepPicturesDir() {
	files, err := ioutil.ReadDir(pc.picturesDir)
	if err != nil {
		glog.Errorf("Cannot read pictures directory '%s'. Skipping photo cleanup", pc.picturesDir)
	}

	now := time.Now()
	aWeekAgo := now.AddDate(0, 0, -7)

	for _, f := range files {
		photoName := f.Name()
		timeOfPhoto, err := pc.photoNameToTime(photoName)
		if err != nil {
			glog.Errorf("Error parsing photo name '%s' to string: %v. Skipping this file", photoName, err)
			continue
		}
		if !pc.inTimeSpan(aWeekAgo, now, *timeOfPhoto) {
			glog.Infof("Deleting photo '%s'", photoName)
			if err := os.Remove(fmt.Sprintf("%s/%s", pc.picturesDir, photoName)); err != nil {
				glog.Errorf("Error deleting photo %s: %v", photoName, err)
			}
		}
	}
}

// Start starts photo cleaner process. Exits when stop is received
func (pc *PhotoCleaner) Start() error {
	pc.timer = time.NewTicker(pc.interval * time.Second)
	for {
		select {
		case time := <-pc.timer.C:
			glog.Infof("Tick at %v. Searching for old pictures", time)
			pc.sweepPicturesDir()
		case <-pc.stop_chan:
			glog.Info("Received stop sign")
			return nil
		}
	}
}

// Stop stops photo cleaner process
func (pc *PhotoCleaner) Stop() error {
	pc.timer.Stop()
	glog.Info("Timer stopped")
	glog.Info("Sending stop sign")
	pc.stop_chan <- 1
	defer pc.wg.Done()
	return nil
}
