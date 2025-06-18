package config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	Receivers  ReceiversConfig  `yaml:"receivers"`
	Exporters  ExportersConfig  `yaml:"exporters"`
	Processors ProcessorsConfig `yaml:"processors"`
	Extensions ExtensionsConfig `yaml:"extensions"`
	Service    ServiceConfig    `yaml:"service"`
}

// ReceiversConfig contains the receiver configurations
type ReceiversConfig struct {
	OTLP *OTLPReceiverConfig `yaml:"otlp"`
}

// OTLPReceiverConfig represents OTLP receiver configuration
type OTLPReceiverConfig struct {
	Protocols ProtocolsConfig `yaml:"protocols"`
}

// ProtocolsConfig contains protocol-specific configurations
type ProtocolsConfig struct {
	GRPC *GRPCConfig `yaml:"grpc"`
	HTTP *HTTPConfig `yaml:"http"`
}

// GRPCConfig represents gRPC protocol configuration
type GRPCConfig struct {
	Endpoint string `yaml:"endpoint"`
}

type HTTPConfig struct {
	Endpoint string `yaml:"endpoint"`
}

// ExportersConfig contains the exporter configurations
type ExportersConfig struct {
	OTLP     *OTLPExporterConfig `yaml:"otlp"`
	OTLPHTTP *OTLPExporterConfig `yaml:"otlphttp"`
}

// OTLPExporterConfig represents OTLP exporter configuration
type OTLPExporterConfig struct {
	Endpoint    string    `yaml:"endpoint"`
	TLS         TLSConfig `yaml:"tls"`
	Compression string    `yaml:"compression"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Insecure bool `yaml:"insecure"`
}

// ProcessorsConfig contains processor configurations
type ProcessorsConfig struct {
	Batch map[string]interface{} `yaml:"batch"`
}

// ExtensionsConfig contains the extension configurations
type ExtensionsConfig struct {
	PProf PProfConfig `yaml:"pprof"`
}

// PProfConfig represents the pprof extension configuration
type PProfConfig struct {
	SaveToFile string `yaml:"save_to_file"`
}

// ServiceConfig contains the service configuration
type ServiceConfig struct {
	Extensions []string            `yaml:"extensions"`
	Pipelines  map[string]Pipeline `yaml:"pipelines"`
}

// Pipeline represents a processing pipeline configuration
type Pipeline struct {
	Receivers  []string `yaml:"receivers"`
	Processors []string `yaml:"processors"`
	Exporters  []string `yaml:"exporters"`
}

func ReadConfig(file string) Config {
	// Read YAML file
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	// Parse YAML into Config struct
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Error parsing YAML: %v", err)
	}

	return config
}

