# EC2 Services

Infrastructure running on `ec2-32-192-103-90.compute-1.amazonaws.com`.

## Services

| Service | Port | User | Description |
|---------|------|------|-------------|
| `my-api-gateway` | 9090 | root | API gateway |
| `upstream` | — | root | Upstream test server |
| `caddy` | 80/443 | — | Reverse proxy / TLS termination |
| `prometheus` | 9091 | prometheus | Metrics collection (scrapes gateway) |
| `grafana-server` | 3000 | grafana | Dashboards (not publicly exposed) |

## SSH access

```bash
ssh -i jit-server.pem ec2-user@ec2-32-192-103-90.compute-1.amazonaws.com
```

## Deploying

Build and deploy all binaries, config, and service files:

```bash
make deploy
```

This cross-compiles for Linux/amd64, SCPs everything to the instance, reloads systemd, and restarts `my-api-gateway` + `upstream`.

## Accessing Grafana

Grafana is only accessible via SSH tunnel (port 3000 is not exposed in security groups):

```bash
ssh -i jit-server.pem -L 3000:localhost:3000 ec2-user@ec2-32-192-103-90.compute-1.amazonaws.com
```

Then open [http://localhost:3000](http://localhost:3000) in your browser.

Default credentials: `admin` / `admin` (change on first login).

Prometheus is pre-configured as the default data source via provisioning (`grafana/provisioning/datasources/prometheus.yaml`).

## Accessing Prometheus

Prometheus is also not publicly exposed. Use an SSH tunnel:

```bash
ssh -i jit-server.pem -L 9091:localhost:9091 ec2-user@ec2-32-192-103-90.compute-1.amazonaws.com
```

Then open [http://localhost:9091](http://localhost:9091).

## Service management

```bash
# Check status
sudo systemctl status my-api-gateway upstream prometheus grafana-server

# Restart a service
sudo systemctl restart my-api-gateway

# View logs
sudo journalctl -u my-api-gateway -f
```

## Load testing

The `loadgen` binary is deployed to `/usr/local/bin/loadgen` on the instance:

```bash
loadgen -url http://localhost:9090/api/test -n 1000 -c 20
loadgen -url http://localhost:9090/api/test -duration 30s -c 10 -rate 50
```
