package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gocql/gocql"
	"github.com/golang/glog"

	"github.com/hydro-monitor/node/pkg/envconfig"
)

// Server represents a server for a specific node
type Server struct {
	client                         *http.Client
	nodeName                       string
	getNodeConfigurationURL        string
	postNodeMeasurementURL         string
	postNodePictureURL             string
	getManualMeasurementRequestURL string
}

// NewServer creates and returns a server taking nodeName and urls from env config
func NewServer() *Server {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	env := envconfig.New()

	return &Server{
		client:                         client,
		nodeName:                       env.NodeName,
		getNodeConfigurationURL:        env.GetNodeConfigurationURL,
		postNodeMeasurementURL:         env.PostNodeMeasurementURL,
		postNodePictureURL:             env.PostNodePictureURL,
		getManualMeasurementRequestURL: env.GetManualMeasurementRequestURL,
	}
}

// State represents a state of a configuration
type State struct {
	// Time interval between measurements in seconds
	// Intervalo de tiempo entre mediciones en segundos
	Interval    int
	// Minimum water level limit to be in this state
	// Límite de nivel de agua para pasar al estado anterior
	UpperLimit  float64
	// Maximum water level to be in this state
	// Límite de nivel de agua para pasar al estado siguiente
	LowerLimit  float64
	// Amount of pictures taken per measurement
	// Cantidad de fotos tomadas por medición
	PicturesNum int
}

// APIConfigutation represents a node configuration response from the hydro monitor server
type APIConfigutation struct {
	States map[string]State `json:"states,inline"`
}

// APIMeasurement represents a measurement creation request for the hydro monitor server
type APIMeasurement struct {
	Time          time.Time `json:"timestamp"`
	WaterLevel    float64   `json:"waterLevel"`
	ManualReading bool      `json:"manualReading"`
}

// APIMeasurementResponse represents a measurement creation response from the hydro monitor server
type APIMeasurementResponse struct {
	APIMeasurement `json:",inline"`
	ReadingID      gocql.UUID `json:"readingId"`
}

// APIPicture represents a picture creation request for the hydro monitor server
type APIPicture struct {
	MeasurementID gocql.UUID `json:"measurementId"`
	Picture       string     `json:"picture"`
	PictureNumber int        `json:"pictureNumber"`
}

// APIMeasurementRequest represents a manual measurement request response from the hydro monitor server
type APIMeasurementRequest struct {
	ManualReading bool `json:"manualReading"`
}

// GetNodeConfiguration returns node configuration from hydro monitor server
func (s *Server) GetNodeConfiguration() (*APIConfigutation, error) {
	resp, err := s.client.Get(fmt.Sprintf(s.getNodeConfigurationURL, s.nodeName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("Node has no configuration loaded")
	}

	statesMap := make(map[string]State)
	err = json.NewDecoder(resp.Body).Decode(&statesMap)
	respConfig := APIConfigutation{
		States: statesMap,
	}
	return &respConfig, err
}

// PostNodeMeasurement sends new measurement to hydro monitor server
func (s *Server) PostNodeMeasurement(measurement APIMeasurement) (*gocql.UUID, error) {
	requestByte, _ := json.Marshal(measurement)
	requestReader := bytes.NewReader(requestByte)
	res, err := s.client.Post(fmt.Sprintf(s.postNodeMeasurementURL, s.nodeName), "application/json", requestReader)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		glog.Errorf("Error reading response body for measurement creation: %v", err)
		return nil, err
	}
	bodyString := string(bodyBytes)
	glog.Infof("Status code for measurement creation: %d. Body: %v", res.StatusCode, bodyString)

	var resObj APIMeasurementResponse
	if err := json.Unmarshal(bodyBytes, &resObj); err != nil {
		glog.Errorf("Error unmarshaling body %v", err)
		return nil, err
	}

	glog.Infof("Returning measurement ID: %v", &resObj.ReadingID)
	return &resObj.ReadingID, nil
}

// PostNodePicture sends new measurement picture to hydro monitor server
func (s *Server) PostNodePicture(measurement APIPicture) error {
	measurementID := measurement.MeasurementID
	picturePath := measurement.Picture

	file, err := os.Open(picturePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("pictureNumber", fmt.Sprintf("%d", measurement.PictureNumber)); err != nil {
		return err
	}

	part, err := writer.CreateFormFile("picture", filepath.Base(picturePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	contentType := writer.FormDataContentType()
	res, err := http.Post(fmt.Sprintf(s.postNodePictureURL, s.nodeName, measurementID), contentType, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		glog.Errorf("Error reading response body for picture upload: %v", err)
		return err
	}
	bodyString := string(bodyBytes)
	glog.Infof("Status code for picture upload: %d. Body: %v", res.StatusCode, bodyString)

	return nil
}

// GetManualMeasurementRequest returns true if manual measurement is requested
func (s *Server) GetManualMeasurementRequest() (bool, error) {
	resp, err := s.client.Get(fmt.Sprintf(s.getManualMeasurementRequestURL, s.nodeName))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	respMeasurementReq := APIMeasurementRequest{}
	if err := json.NewDecoder(resp.Body).Decode(&respMeasurementReq); err != nil {
		return false, err
	}
	if respMeasurementReq.ManualReading {
		return true, nil
	}
	return false, nil
}
