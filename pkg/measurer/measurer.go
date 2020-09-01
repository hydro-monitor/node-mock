package measurer

import (
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/hydro-monitor/node/pkg/camera"
	"github.com/hydro-monitor/node/pkg/envconfig"
	"github.com/hydro-monitor/node/pkg/server"
	"github.com/hydro-monitor/node/pkg/water"
)

// Measurer represents a measurer
type Measurer struct {
	index                            int
	measurements                     []float64
	trigger_chan                     chan int
	manual_chan                      chan int
	analyzer_chan                    chan float64
	stop_chan                        chan int
	wg                               *sync.WaitGroup
	waterLevel                       *water.WaterLevel
	camera                           *camera.Camera
	server                           *server.Server
	measurementToAnalyzerSendTimeout time.Duration
}

// NewMeasurer creates and returns a new measurer
func NewMeasurer(trigger_chan, manual_chan chan int, analyzer_chan chan float64, wg *sync.WaitGroup) *Measurer {
	env := envconfig.New()
	return &Measurer{
		index:                            0,
		measurements:                     []float64{1,3,4,5,6}, // FIXME make me an env variable
		trigger_chan:                     trigger_chan,
		analyzer_chan:                    analyzer_chan,
		manual_chan:                      manual_chan,
		stop_chan:                        make(chan int),
		wg:                               wg,
		server:                           server.NewServer(),
		measurementToAnalyzerSendTimeout: time.Duration(env.MeasurementToAnalyzerSendTimeout) * time.Second,
	}
}

// takePicture takes a new picture with camera. Uses time as picture name
func (m *Measurer) takePicture(time time.Time) (string, error) {
	return "/assets/photo.jpeg", nil
}

// takeWaterLevelMeasurement takes water level with water level module
func (m *Measurer) takeWaterLevelMeasurement() float64 {
	f := m.measurements[m.index % len(m.measurements)]
	m.index++

	glog.Infof("Sending measurement %f to analyzer", f)
	select {
	case m.analyzer_chan <- f:
		glog.Info("Measurement sent")
	case <-time.After(m.measurementToAnalyzerSendTimeout):
		glog.Warning("Measurement send timed out")
	}

	return f
}

// takeMeasurement takes water level, sends water measurement to server. 
// Takes picture and uploads picture to new server measurement.
func (m *Measurer) takeMeasurement(manual bool) {
	time := time.Now()

	glog.Info("Taking water level")
	waterLevel := m.takeWaterLevelMeasurement()

	glog.Infof("Sending measurement (water level: %f and picture) to server", waterLevel)
	measurementID, err := m.server.PostNodeMeasurement(server.APIMeasurement{
		Time:       time,
		WaterLevel: waterLevel,
		ManualReading:  manual,
	})
	if err != nil {
		glog.Errorf("Error sending measurement %f to server: %v. Skipping measurement", waterLevel, err)
		return
	}

	glog.Info("Taking picture")
	go func() {
		pictureFile, err := m.takePicture(time)
		if err != nil {
			glog.Errorf("Error taking picture: %v. Skipping measurement", err)
			return
		}

		if err := m.server.PostNodePicture(server.APIPicture{
			MeasurementID: *measurementID,
			Picture:       pictureFile,
			PictureNumber: 1, // TODO Pending implementation of multiple pictures per measurement
		}); err != nil {
			glog.Errorf("Error sending picture to server: %v", err)
			return
		}
	}()
}

// Start starts measurer process. Exits when stop is received
func (m *Measurer) Start() error {
	defer m.wg.Done()
	for {
		select {
		case <-m.trigger_chan:
			glog.Info("Received alert from Trigger. Requesting measurement")
			m.takeMeasurement(false)
		case <-m.manual_chan:
			glog.Info("Received alert from ManualTrigger. Requesting measurement")
			m.takeMeasurement(true)
		case <-m.stop_chan:
			glog.Info("Received stop sign")
			return nil
		}
	}
}

// Stop stops measurer process
func (m *Measurer) Stop() error {
	glog.Info("Sending stop sign")
	m.stop_chan <- 1
	return nil
}
