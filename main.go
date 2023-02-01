// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/is-consul-health-agent/health"
	log "github.com/sirupsen/logrus"
)

var (
	consulClient   *api.Client
	clusterSize    int
	isBootstrapped bool = false
	mu             sync.Mutex
)

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if !isBootstrapped {
		mu.Lock() // We only mutate state in the initial health check, it's a one-way street
		defer mu.Unlock()

		// Check again once the lock has been acquired. If a concurrent check
		// has validated cluster state, we can short-circuit.
		if isBootstrapped {
			w.WriteHeader(http.StatusOK)
			return
		}

		check := health.BootstrapHealthCheck{
			Client:      consulClient,
			ClusterSize: clusterSize,
		}
		if check.IsHealthy(r.Context()) {
			// Change global state
			// Once this has succeeded, we switch to a standard health check
			log.Info("Cluster bootstrapping succeeded! Switching to standard health check.")
			isBootstrapped = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	check := health.NodeHealthCheck{
		Client: consulClient,
	}
	if check.IsHealthy() {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

func main() {
	var (
		err        error
		addr       string
		token      string
		listenAddr string
	)

	const (
		envClusterSize  = "CONSUL_CLUSTER_SIZE"
		envClusterAddr  = "CONSUL_HTTP_ADDR"
		envClusterToken = "CONSUL_HTTP_TOKEN"
		envListenerPort = "CONSUL_HEALTH_PORT"
	)

	if rawAddr, ok := os.LookupEnv(envClusterAddr); ok {
		addr = rawAddr
	} else {
		addr = "http://127.0.0.1:8500"
		log.Warnf("%s was not set, defaulting to %s", envClusterAddr, addr)
	}

	if rawToken, ok := os.LookupEnv(envClusterToken); ok {
		token = rawToken
	} else {
		log.Warnf("%s was not set, defaulting to no token.", envClusterToken)
	}

	if rawAddr, ok := os.LookupEnv(envListenerPort); ok {
		listenAddr = rawAddr
	} else {
		log.Warnf("%s was not set, defaulting to 8080", envListenerPort)
		listenAddr = "8080"
	}
	listenAddr = fmt.Sprintf(":%s", listenAddr)

	if raw, ok := os.LookupEnv(envClusterSize); ok {
		i, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			log.Fatal("set cluster size: ", err)
		}
		clusterSize = int(i)
	} else {
		log.Fatalf("%s was not set. Exiting.", envClusterSize)
	}

	consulClient, err = api.NewClient(&api.Config{
		Address: addr,
		Token:   token,
	})
	if err != nil {
		log.Fatal("initialize Consul client: ", err)
	}

	log.Info("Starting up health check listener")

	http.HandleFunc("/health", healthCheckHandler)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatal("HTTP server exited: ", err)
	}
}
