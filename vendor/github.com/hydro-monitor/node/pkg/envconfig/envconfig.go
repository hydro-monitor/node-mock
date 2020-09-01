package envconfig

import (
	"os"
	"strconv"
	"strings"
)

// Config represents the config for the node
type Config struct {
	// Nombre del nodo
	NodeName                            string
	// Altura a la cual se encuentra el sensor, medida desde el fondo del río en cm
	WaterSensorDistance                 int
	// Intervalo inicial entre toma de mediciones. Es el intervalo de tiempo en segundos 
	// entre cada toma de medición hasta cargar la configuración del nodo presente en el servidor
	InitialTriggerInterval              int
	// Intervalo entre chequeo de actualización de configuración. Es el intervalo de tiempo
	// en segundos entre cada chequeo de actualización de configuración del nodo.
	ConfigurationUpdateInterval         int
	// Intervalo entre chequeo de pedido de medición manual. Es el intervalo de tiempo en segundos
	// entre cada chequeo de pedido de medición manual.
	ManualMeasurementPollInterval       int
	// Intervalo entre limpieza de fotos de picturesDir. Es el intervalo de tiempo en horas
	// entre cada limpieza de fotos de picturesDir.
	PhotoCleaningInterval               int
	// Directorio donde se guardan las fotos tomadas por la cámara
	PicturesDir                         string
	// Puerto serie de Arduino
	SerialPort                          string
	// Baud rate de Arduino
	Baud                                int
	// URL servidor
	ServerURL                           string
	// URL para obtener configuración de nodo
	GetNodeConfigurationURL             string
	// URL para crear medición
	PostNodeMeasurementURL              string
	// URL para subir foto de una medición
	PostNodePictureURL                  string
	// URL para obtener pedido de medición manual
	GetManualMeasurementRequestURL      string
	// Timeout para actualización de intervalo de toma de mediciones
	IntervalUpdateTimeout               int
	// Timeout para actualización de configuración de nodo
	ConfigurationUpdateTimeout          int
	// Timeout para pedido de medición manual
	ManualMeasurementRequestSendTimeout int
	// Timeout de envío de nueva medición para analizar 
	MeasurementToAnalyzerSendTimeout    int
}

// New returns a new Config struct loading variables from .env and using defaults for the values not present
func New() *Config {
	serverURL := getEnv("SERVER_URL", "http://antiguos.fi.uba.ar:443")

	return &Config{
		NodeName:                            getEnv("NODE_NAME", "1"),
		WaterSensorDistance:                 getEnvAsInt("WATER_SENSOR_DISTANCE", 600),
		InitialTriggerInterval:              getEnvAsInt("INITIAL_TRIGGER_INTERVAL", 10),
		ConfigurationUpdateInterval:         getEnvAsInt("CONFIGURATION_UPDATE_INTERVAL", 60),
		ManualMeasurementPollInterval:       getEnvAsInt("MANUAL_MEASUREMENT_POLL_INTERVAL", 180),
		PhotoCleaningInterval:               getEnvAsInt("PHOTO_CLEANING_INTERVAL", 72),
		PicturesDir:                         getEnv("PICTURES_DIR", "/home/pi/Documents/pictures"),
		SerialPort:                          getEnv("SERIAL_PORT", "/dev/ttyACM0"),
		Baud:                                getEnvAsInt("BAUD", 9600),
		ServerURL:                           serverURL,
		GetNodeConfigurationURL:             serverURL + getEnv("GET_NODE_CONFIGURATION_PATH", "/api/nodes/%s/configuration"),
		PostNodeMeasurementURL:              serverURL + getEnv("POST_NODE_MEASUREMENT_PATH", "/api/nodes/%s/readings"),
		PostNodePictureURL:                  serverURL + getEnv("POST_NODE_PICTURE_PATH", "/api/nodes/%s/readings/%s/photos"),
		GetManualMeasurementRequestURL:      serverURL + getEnv("GET_MANUAL_MEASUREMENT_REQUEST_PATH", "/api/nodes/%s/manual-reading"),
		IntervalUpdateTimeout:               getEnvAsInt("INTERVAL_UPDATE_TIMEOUT", 10),
		ConfigurationUpdateTimeout:          getEnvAsInt("CONFIGURATION_UPDATE_TIMEOUT", 10),
		ManualMeasurementRequestSendTimeout: getEnvAsInt("MANUAL_MEASUREMENT_REQUEST_SEND_TIMEOUT", 10),
		MeasurementToAnalyzerSendTimeout:    getEnvAsInt("MEASUREMENT_TO_ANALYZER_SEND_TIMEOUT", 10),
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

// Simple helper function to read an environment variable into integer or return a default value
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}

// Helper to read an environment variable into a bool or return default value
func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}

	return defaultVal
}

// Helper to read an environment variable into a string slice or return default value
func getEnvAsSlice(name string, defaultVal []string, sep string) []string {
	valStr := getEnv(name, "")

	if valStr == "" {
		return defaultVal
	}

	val := strings.Split(valStr, sep)

	return val
}
