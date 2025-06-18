package main

import (
	"fmt"
	"github.com/streamfold/fluentbit-otel-wrapper/internal/config"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	//"slices"
	"strings"
)

type dbgFileType interface {
	io.WriteCloser
	io.StringWriter
}

func main() {
	// Get all command line arguments
	args := os.Args

	if len(args) <= 2 {
		log.Fatalf("Usage: fluentbit-otel-wrapper --config <path to config>")
	}

	fluentBitPath := os.Getenv("FLUENTBIT_PATH")

	// Get the ROTEL_PATH environment variable
	if fluentBitPath == "" {
		log.Fatalf("ROTEL_PATH environment variable not set")
	}

	// Check for --config argument and process the config file
	configPath := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			configPath = args[i+1]
			break
		}
	}

	dbgFile := os.Getenv("FLUENTBIT_WRAPPER_DEBUG_FILE")
	var df dbgFileType

	// Open the file for appending (create it if it doesn't exist)
	if dbgFile != "" {
		f, err := os.OpenFile(dbgFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Error opening file %s: %v", dbgFile, err)
		}
		df = f

		defer func() {
			if df != nil {
				_ = df.Close()
			}
		}()
	} else {
		df = nopWriteCloser{io.Discard}
	}

	// Log all arguments to the file
	argsStr := strings.Join(args, " ")
	if _, err := df.WriteString(argsStr + "\n"); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	if configPath == "" {
		log.Fatalf("Can not find --config path")
	}

	// Read the config file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// Append the config file contents to args.out
	if _, err := df.WriteString("--- Config file contents ---\n"); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
	if _, err := df.Write(configData); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
	if _, err := df.WriteString("\n--- End of config file ---\n"); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	cmdArgs := []string{"-c", "fluent-bit.conf"}

	conf := config.ReadConfig(configPath)
	file, err := os.OpenFile("fluent-bit.conf", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close() // Ensures the file is closed when done.
	file.WriteString("[SERVICE]\n")
	file.WriteString("\t\tflush 1\n")
	file.WriteString("\t\tdaemon Off\n")
	file.WriteString("\t\tlog_level info\n")

	if grpc := conf.Receivers.OTLP.Protocols.GRPC; grpc != nil {
		file.WriteString("[INPUT]\n")
		file.WriteString("\t\tname opentelemetry\n")
		file.WriteString("\t\tlisten 127.0.0.1\n")
		parts := strings.Split(grpc.Endpoint, ":")
		file.WriteString("\t\tport " + parts[1] + "\n")
	} else if http := conf.Receivers.OTLP.Protocols.HTTP; http != nil {
		file.WriteString("[INPUT]\n")
		file.WriteString("\t\tname opentelemetry\n")
		file.WriteString("\t\tlisten 127.0.0.1\n")
		parts := strings.Split(http.Endpoint, ":")
		file.WriteString("\t\tport " + parts[1] + "\n")
	} else {
		log.Fatalf("can not find receiver configuration")
	}

	if otlp := conf.Exporters.OTLP; otlp != nil {
		file.WriteString("[OUTPUT]\n")
		file.WriteString("\t\tname opentelemetry\n")
		ep := endpointWithScheme(otlp)
		ep = strings.TrimPrefix(ep, "http://")
		parts := strings.Split(ep, ":")
		file.WriteString("\t\thost " + parts[0] + "\n")
		file.WriteString("\t\tport " + parts[1] + "\n")
		file.WriteString("\t\tgrpc on\n")
		file.WriteString("\t\tmetrics_uri /v1/metrics\n")
		file.WriteString("\t\tlogs_uri /v1/logs\n")
		file.WriteString("\t\ttraces_uri /v1/traces\n")
		if otlp.Compression == "gzip" {
			file.WriteString("\t\tcompress gzip\n")
		}
	} else if otlphttp := conf.Exporters.OTLPHTTP; otlphttp != nil {
		file.WriteString("[OUTPUT]\n")
		file.WriteString("\t\tname opentelemetry\n")
		ep := endpointWithScheme(otlphttp)
		parts := strings.Split(ep, ":")
		file.WriteString("\t\thost " + parts[0] + "\n")
		file.WriteString("\t\tport " + parts[1] + "\n")
		file.WriteString("\t\thttp2 Off\n")
		file.WriteString("\t\tmetrics_uri /v1/metrics\n")
		file.WriteString("\t\tlogs_uri /v1/logs\n")
		file.WriteString("\t\ttraces_uri /v1/traces\n")
		if otlphttp.Compression == "gzip" {
			file.WriteString("\t\tcompress gzip\n")
		}
	} else {
		log.Fatalf("can not find exporter configuration")
	}
	//
	// hasBatch := false
	// for _, pipeline := range conf.Service.Pipelines {
	// 	// XXX: There's usually only one pipeline, but check if any have a batch processor
	// 	if slices.Contains(pipeline.Processors, "batch") {
	// 		hasBatch = true
	// 	}
	// }
	//
	// // If there's no batch, disable it
	// if !hasBatch {
	// 	cmdArgs = append(cmdArgs, "--disable-batching")
	// }

	programName := filepath.Base(fluentBitPath)

	// Create command for execution
	binary, lookErr := exec.LookPath(fluentBitPath)
	if lookErr != nil {
		log.Fatalf("Error finding executable: %v\n", lookErr)
	}

	execArgs := []string{programName}
	execArgs = append(execArgs, cmdArgs...)

	_, _ = df.WriteString(fmt.Sprintf("Executing rotel at: %v (%v)\n", fluentBitPath, execArgs))

	_ = df.Close()
	df = nil

	execErr := syscall.Exec(binary, execArgs, os.Environ())
	if execErr != nil {
		log.Fatalf("Error executing %s: %v\n", fluentBitPath, execErr)
	}
}

func endpointWithScheme(otlp *config.OTLPExporterConfig) string {
	endpoint := otlp.Endpoint
	if strings.HasPrefix(endpoint, "http://") {
		return strings.TrimPrefix(endpoint, "http://")
	}

	if strings.HasPrefix(endpoint, "https://") || !otlp.TLS.Insecure {
		panic("https endpoints not supported")
	}

	return fmt.Sprintf("http://%s", endpoint)
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func (nopWriteCloser) WriteString(s string) (n int, err error) {
	return len(s), nil
}
