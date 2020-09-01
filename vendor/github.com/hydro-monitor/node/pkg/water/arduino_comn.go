package water

import (
	"time"

	"github.com/golang/glog"
	"github.com/tarm/serial"

	"github.com/hydro-monitor/node/pkg/envconfig"
)

// ArduinoCommunicator represents an Arduino communicator 
type ArduinoCommunicator struct {
	port *serial.Port
}

// NewArduinoCommunicator creates and returns a new Arduino communicator taking serial port and baud rate from envconfig.
// Opens the serial port and waits for Arduino to boot.
func NewArduinoCommunicator() *ArduinoCommunicator {
	env := envconfig.New()

	c := &serial.Config{
		Name: env.SerialPort,
		Baud: env.Baud,
	}
	s, err := serial.OpenPort(c)
	if err != nil {
		glog.Fatalf("Error opening serial port %v", err)
	}

	// We need to sleep the program for 2 seconds because every time a new
	// serial connection is made with the Arduino it resets similar to when
	// you are uploading your program to it.
	time.Sleep(2 * time.Second)

	return &ArduinoCommunicator{
		port: s,
	}
}

// RequestMeasurement writes a byte on serial port to ask Arduino for a new measurement
func (ac *ArduinoCommunicator) RequestMeasurement() error {
	req := []byte{1}
	// Write will block until at least one byte is written
	_, err := ac.port.Write(req)
	if err != nil {
		glog.Errorf("Error writing to serial port %v", err)
		return err
	}
	return nil
}

// read reads bytes written on serial port
func (ac *ArduinoCommunicator) read(buffer []byte) (int, error) {
	n, err := ac.port.Read(buffer)
	if err != nil {
		glog.Errorf("Error reding from serial port %v", err)
		return n, err
	}
	return n, nil
}

// ReadMeasurement reads measurement written on serial port and returns the amount of bytes read
func (ac *ArduinoCommunicator) ReadMeasurement(buffer []byte) (int, error) {
	// Read will block until at least one byte is returned
	n, err := ac.read(buffer)
	if err != nil {
		return n, err
	}
	glog.Infof("Data received is: %q", buffer[:n])

	for buffer[n-1] != '\n' {
		nTmp, err := ac.read(buffer[n:])
		if err != nil {
			return n + nTmp, err
		}
		n = n + nTmp
	}

	glog.Infof("Measurement received is: %q", buffer[:n])
	return n, nil
}

// Close closes serial port
func (ac *ArduinoCommunicator) Close() error {
	if err := ac.port.Close(); err != nil {
		glog.Errorf("Error closing serial port %v", err)
		return err
	}
	return nil
}
