package main

import (
	"net/http"

	"github.com/hashicorp/is-consul-health-agent/health"
	log "github.com/sirupsen/logrus"
)

var isBootstrapped bool = false

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if !isBootstrapped {
		check := health.BootstrapHealthCheck{
			Client:      consulClient,
			ClusterSize: clusterSize,
		}
		if check.IsHealthy(r.Context()) {
			// Change global state
			// Once this has succeeded, we switch to a standard health check
			log.Info("Cluster bootstrapping succeeded! Switching to standard health response.")
			isBootstrapped = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
