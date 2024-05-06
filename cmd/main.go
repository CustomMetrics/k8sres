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

var port string = "9805"

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}

	rConfig, err := config.GetConfig()
	if err != nil {
		log.Error("error in getting KUBECONFIG %v", err)
		return
	}

	err = krmetrics.SetClients(rConfig)
	if err != nil {
		log.Fatalf("Error building extra clientset: %s", err.Error())
	}

	err = krmetrics.SetCollectors(supCfg.K8sResourceMetricYaml)
	if err != nil {
		log.Error("error setting resource to metrics collector", err.Error())
	}
	log.Info("Start K8s res Metrics Server")
	server := http.NewServeMux()
	// Handle the endpoint serving the metrics
	server.Handle("/metrics", promhttp.Handler())
	// Create a server object
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: server,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting nvc exporter: %s\n", err)
	}
}
