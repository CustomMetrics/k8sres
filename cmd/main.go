package main

import (
	"os"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	krmetrics "github.com/CustomMetrics/k8sres/pkg/k8sresourcemetrics"
	supCfg "github.com/CustomMetrics/k8sres/resconfig"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Default port number where the server will listen
var port string = "9805"

func main() {
	// Setting up JSON formatter for logging
	log.SetFormatter(&log.JSONFormatter{})
	// Setting log output to standard output
	log.SetOutput(os.Stdout)

	// Check for a DEBUG environment variable to set logging level
	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}

	// Get Kubernetes configuration from the default location
	rConfig, err := config.GetConfig()
	if err != nil {
		log.Error("error in getting KUBECONFIG %v", err)
		return
	}

	// Initialize Kubernetes clients using the obtained config
	err = krmetrics.SetClients(rConfig)
	if err != nil {
		log.Fatalf("Error building extra clientset: %s", err.Error())
	}

	// Set up metric collectors from a predefined configuration file
	err = krmetrics.SetCollectors(supCfg.K8sResourceMetricYaml)
	if err != nil {
		log.Error("error setting resource to metrics collector", err.Error())
	}
	// Log that the server is starting
	log.Info("Start K8s res Metrics Server")

	// Create a new HTTP ServeMux (router)
	server := http.NewServeMux()
	// Handle the "/metrics" endpoint with Prometheus's handler to expose the metrics
	server.Handle("/metrics", promhttp.Handler())

	// Setting up the HTTP server configuration
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: server,
	}

	// Start listening and serving HTTP requests
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting nvc exporter: %s\n", err)
	}
}
