package health

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

// BootstrapHealthCheck contains the resources and methods necessary to
// confirm an autopilot-enabled Consul cluster has reached a healthy
// state and assumed cluster leadership for a given Upgrade Version Tag
type BootstrapHealthCheck struct {
	Client      *api.Client
	ClusterSize int
}

type consulServer struct {
	health api.ServerHealth
	node   *api.Node
}

// IsHealthy analyzes a Consul cluster to determine several things:
// 1. Is the cluster reporting a healthy, internally consistent state?
// 2. Does the cluster have the expected number of nodes?
// 3. Are all nodes in the current Upgrade Version Tag group voting?
// 4. Is the cluster leader a member of the current Upgrade Version Tag group?
// 5. Have all remaining servers in the cluster released voter status?
func (hc *BootstrapHealthCheck) IsHealthy() bool {
	clusterVersion, err := hc.getLocalClusterVersion()
	if err != nil {
		log.Error("Unable to determine value of Autopilot UpgradeVersionTag: ", err)
		return false
	}

	ready, err := hc.isClusterReady(clusterVersion)
	if err != nil {
		log.Error(err)
		return false
	}

	return ready
}

func (hc *BootstrapHealthCheck) getLocalClusterVersion() (string, error) {
	agent := hc.Client.Agent()
	agentStatus, err := agent.Self()
	if err != nil {
		return "", fmt.Errorf("Unable to retrieve Agent Self status: %w", err)
	}

	if version, ok := agentStatus["Meta"]["consul_cluster_version"]; ok {
		return version.(string), nil
	}

	return "", errors.New("Autopilot Upgade Version Tag missing from node configuration")
}

func (hc *BootstrapHealthCheck) isClusterReady(clusterVersion string) (bool, error) {
	//////////////////////////////////////
	// Build a list of Consul servers
	//////////////////////////////////////
	var (
		oldServers []consulServer
		newServers []consulServer
	)

	autopilot, err := hc.Client.Operator().AutopilotServerHealth(&api.QueryOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "429") {
			log.Info("Autopilot reports cluster is not yet in a healthy state")
			return false, nil
		}

		if strings.Contains(err.Error(), "No cluster leader") {
			log.Info("Cluster has no leader")
			return false, nil
		}

		return false, fmt.Errorf("Error querying autopilot health: %w", err)
	}

	if !autopilot.Healthy {
		// This is caught above in the 429 error
		log.Info("Autopilot reports cluster is not yet in a healthy state")
		return false, nil
	}

	filters := make([]string, len(autopilot.Servers))
	serversByID := make(map[string]api.ServerHealth)

	for i, server := range autopilot.Servers {
		filters[i] = fmt.Sprintf(`ID == "%s"`, server.ID)
		serversByID[server.ID] = server
	}

	opt := &api.QueryOptions{
		Filter: strings.Join(filters, " or "),
	}
	nodes, _, err := hc.Client.Catalog().Nodes(opt)
	if err != nil {
		return false, fmt.Errorf("Error querying Consul Catalog Nodes: %w", err)
	}

	if len(nodes) != len(autopilot.Servers) {
		log.Info("Autopilot and Catalog report a different number of nodes. Waiting for consistency...")
		return false, nil
	}

	for _, node := range nodes {
		health, ok := serversByID[node.ID]
		if !ok {
			return false, fmt.Errorf("Node discovered in Catalog, but not matched by Autopilot: %s", node.ID)
		}

		s := consulServer{
			health: health,
			node:   node,
		}

		if node.Meta["consul_cluster_version"] == clusterVersion {
			newServers = append(newServers, s)
		} else {
			oldServers = append(oldServers, s)
		}
	}

	//////////////////////////////////////
	// Fail if we have not reached the
	// expected node count
	//////////////////////////////////////
	if len(newServers) < hc.ClusterSize {
		log.Infof("Number of new servers does not yet match expected cluster size. Total Nodes: %d Expected Nodes: %d", len(newServers), hc.ClusterSize)
		return false, nil
	}

	//////////////////////////////////////
	// Fail if autopilot has not fully
	// transferred voter status
	//////////////////////////////////////
	var foundLeader bool
	for _, server := range newServers {
		if !server.health.Voter {
			log.Infof("Replacement node is not yet voting: %s", server.health.ID)
			return false, nil
		}

		if server.health.Leader {
			foundLeader = true
		}
	}

	//////////////////////////////////////
	// Fail if we didn't see a leader in
	// the newservers set
	//////////////////////////////////////
	if !foundLeader {
		log.Info("Cluster leadership has not yet transferred to new node set")
		return false, nil
	}

	//////////////////////////////////////
	// Fail if old servers still hold
	// voter status
	//////////////////////////////////////
	for _, server := range oldServers {
		if server.health.Voter {
			log.Info("An outgoing node still holds voter status. Waiting for all old nodes to step down...")
			return false, nil
		}
	}

	return true, nil
}
