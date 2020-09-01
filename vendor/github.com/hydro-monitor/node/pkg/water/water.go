package water

import (
	"strconv"
	"strings"

	"github.com/golang/glog"

	"github.com/hydro-monitor/node/pkg/envconfig"
)

// WaterLevel represents a water level measurer module
type WaterLevel struct {
	comm *ArduinoCommunicator
	sensorDistance float64
}

// NewWaterLevel creates and returns a new water level module taking sensorDistance from envconfig
func NewWaterLevel() *WaterLevel {
	env := envconfig.New()
	return &WaterLevel{
		comm: NewArduinoCommunicator(),
		sensorDistance: float64(env.WaterSensorDistance),
	}
}

// TakeWaterLevel requests a new measurement, reads it, and returns the water level 
// after substracting it from sensorDistance
func (w *WaterLevel) TakeWaterLevel() (float64, error) {
	if err := w.comm.RequestMeasurement(); err != nil {
		glog.Errorf("Error requesting measurement to Arduino %v", err)
		return -1, err
	}

	buffer := make([]byte, 128)
	n, err := w.comm.ReadMeasurement(buffer)
	if err != nil {
		glog.Errorf("Error reading measurement from Arduino %v", err)
		return -1, err
	}
	/*
	buffer := make([]byte, 128) // TODO Remove mock measurement
	buffer[0] = '6'
	buffer[1] = '5'
	
	n := 2
	*/

	glog.Infof("Measurement received: %q", buffer[:n])
	str := string(buffer[:n])
	nStr := strings.TrimRight(str, "\r\n")
	f, err := strconv.ParseFloat(nStr, 64)
	if err != nil {
		glog.Errorf("Failed to convert string '%s' to int: %v", nStr, err)
		return -1, err
	}

	glog.Infof("Substracting measurement from sensor distance: %f - %f", w.sensorDistance, f)
	level := w.sensorDistance - f
	glog.Infof("Resulting water level: %f", level)

	return level, nil
}
