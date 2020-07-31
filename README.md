# Implementation Services - Consul Autopilot Health Check Agent
This repository contains an application designed to execute health checks against a Hashicorp Consul Enterprise cluster. The goal of these health checks is to ensure an Autopilot-enabled cluster utilizing UpgradeVersionTag for automatic blue/green upgrades has successfully transferred voter status and cluster leadership, then output this information for observation using an outside (e.g. cloud platform) health checking service.

The impetus for developing this service lies with deployment in Google Cloud Platform, or anywhere else a signaling mechanism such as EC2 Instance Lifecycle Hooks or Azure VM Extensions does not exist. With the addition of this agent, we can observe and ensure a Consul cluster has fully transitioned voter and leader status to a set of replacement instances, before destroying the outgoing node set.

The health checking logic was derived from the HashiCorp Cloud Platform team's [Consul Host Manager](https://github.com/hashicorp/cloud-consul-host-manager) agent.

## Configuration
A systemd unit file is included in this repository, which will launch the agent and ensure it it restarted in the event of failure. The unitfile alone, however, is not enough to launch the application. It is necessary to provide a [drop-in unit](https://coreos.com/os/docs/latest/using-systemd-drop-in-units.html) to configure environment variables the application relies on to communicate with the Consul cluster.

For example, if the loaded systemd unit is named `consul-health.service`, a drop-in may look like this:

**/etc/systemd/system/consul-health.service.d/10-environment.conf:**
```
[Service]
Environment=CONSUL_HTTP_ADDR=unix:///run/consul_kv/consul_kv_http.sock
Environment=CONSUL_CLUSTER_SIZE=5
Environment=CONSUL_HEALTH_PORT=8080
```

The supported environment variables are as follows:

| Environment Variable | Default               | Description                                                                                                                            |
|----------------------|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------|
| CONSUL_HTTP_ADDR     | http://localhost:8500 | Address at which Consul's HTTP listener may be reached. Prefix with `http://` for TCP transport, or `unix://` for UNIX Domain Sockets. |
| CONSUL_HTTP_TOKEN    | None                  | ACL Token to use when connecting to Consul                                                                                             |
| CONSUL_CLUSTER_SIZE  | None                  | **REQUIRED** Number of instances that should gain voter status before a cluster is considered healthy                                  |
| CONSUL_HEALTH_PORT   | 8080                  | TCP port for the health check's HTTP server to listen on                                                                               |

## Building
```
git clone git@github.com:hashicorp/is-consul-health-agent.git
cd is-consul-health-agent
go build
```
...Profit!

## // TODO:
- [ ] Create a DefaultHealthCheck struct
  - Once a cluster is healthy, forward subsequent health checks to a Consul endpoint (e.g. /operator/autopilot/health).
  - Currently after the cluster has bootstrapped we change state to automatically returning HTTP 200.
- [ ] Add a commander ([mitchellh/cli](https://github.com/mitchellh/cli) or [spf12/cobra](https://github.com/spf13/cobra)).
- [ ] Clean up config/environment variable handling.
- [ ] Add signal handling. Refresh state on SIGHUP.