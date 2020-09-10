package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

var consulClient *api.Client
var clusterSize int

func main() {
	var (
		err        error
		addr       string
		token      string
		listenAddr string
	)

	if addr = os.Getenv("CONSUL_HTTP_ADDR"); addr == "" {
		addr = "http://127.0.0.1:8500"
		log.Warn("CONSUL_HTTP_ADDR was not set, defaulting to http://127.0.0.1:8500")
	}

	if token = os.Getenv("CONSUL_HTTP_TOKEN"); token == "" {
		log.Warn("CONSUL_HTTP_TOKEN was not set, defaulting to no token.")
	}

	if listenAddr = os.Getenv("CONSUL_HEALTH_PORT"); listenAddr == "" {
		log.Warn("CONSUL_HEALTH_PORT was not set, defaulting to 8080")
		listenAddr = "8080"
	}
	listenAddr = fmt.Sprintf(":%s", listenAddr)

	raw := os.Getenv("CONSUL_CLUSTER_SIZE")
	if raw == "" {
		log.Fatal("CONSUL_CLUSTER_SIZE was not set. Exiting.")
		os.Exit(-1)
	} else {
		i, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			log.Fatal("Unable to parse CONSUL_CLUSTER_SIZE")
			os.Exit(-1)
		}
		clusterSize = int(i)
	}

	consulClient, err = api.NewClient(&api.Config{
		Address: addr,
		Token:   token,
	})
	if err != nil {
		log.Fatal("Unable to initialize Consul client: ", err)
		os.Exit(-1)
	}

	log.Info("Starting up health check listener")
	http.HandleFunc("/health", healthCheckHandler)

	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatal("HTTP server exited: ", err)
	}
}
