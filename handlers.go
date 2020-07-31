package main

import (
	"net/http"

	"github.com/hashicorp/is-consul-health-agent/health"
)

var isBootstrapped bool = false

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if !isBootstrapped {
		check := health.BootstrapHealthCheck{
			Client:      consulClient,
			ClusterSize: clusterSize,
		}
		if check.IsHealthy() {
			// Change global state
			// Once this has succeeded, we switch to a standard health check
			isBootstrapped = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
