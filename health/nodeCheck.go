package health

import (
	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

// NodeHealthCheck performs a simple cluster/node sign of life test.
// /v1/status/leader is queried. In the event the node is unable to respond,
// the node is live but there is no Raft leader, or the API otherwise returns
// an error, we evaluate the node as Unehalthy.
type NodeHealthCheck struct {
	Client *api.Client
}

// IsHealthy implements the basic boolean test logic described above
func (hc *NodeHealthCheck) IsHealthy() bool {
	leader, err := hc.Client.Status().Leader()
	if err != nil {
		log.Error("Cluster status returned error: ", err)
		return false
	}
	// If we evaluate this health check on a cluster that has not
	// yet reached bootstrap_expect, this endpoint returns 200.
	// To prevent false positive, we check for empty string as well.
	if leader == "" {
		log.Error("Cluster reported no Raft leader")
		return false
	}
	return true
}
