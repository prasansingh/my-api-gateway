# EC2 Services

Infrastructure running on `ec2-32-192-103-90.compute-1.amazonaws.com`.

**Instance type: t3.small** (2 GB RAM, 2 vCPU). Upgraded from t3.micro (1 GB) which OOM'd under the full service stack. To resize: EC2 Console → Stop instance → Actions → Instance Settings → Change Instance Type → Start.

## Services

| Service | Port | User | Description |
|---------|------|------|-------------|
| `my-api-gateway` | 9090 | root | API gateway |
| `upstream` | — | root | Upstream test server |
| `caddy` | 80/443 | — | Reverse proxy / TLS termination |
| `prometheus` | 9091 | prometheus | Metrics collection (scrapes gateway) |
| `grafana-server` | 3000 | grafana | Dashboards (not publicly exposed) |
| `postgresql` | 5432 | postgres | API key storage for auth middleware |

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

## PostgreSQL

PostgreSQL 17 stores API keys for the gateway's auth middleware. Not publicly exposed — listens on localhost only.

### Setup (one-time)

```bash
sudo dnf install -y postgresql17-server postgresql17
sudo postgresql-setup --initdb
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

Enable password auth (replace default `ident`/`peer` with `md5` in `pg_hba.conf`):

```bash
sudo sed -i 's/^\(local.*all.*all.*\)peer$/\1md5/' /var/lib/pgsql/data/pg_hba.conf
sudo sed -i 's/^\(host.*all.*all.*127.0.0.1\/32.*\)ident$/\1md5/' /var/lib/pgsql/data/pg_hba.conf
sudo sed -i 's/^\(host.*all.*all.*::1\/128.*\)ident$/\1md5/' /var/lib/pgsql/data/pg_hba.conf
sudo systemctl restart postgresql
```

Create the database and user:

```bash
sudo -u postgres psql <<EOF
CREATE USER gateway WITH PASSWORD 'your_password_here';
CREATE DATABASE gateway OWNER gateway;
GRANT ALL PRIVILEGES ON DATABASE gateway TO gateway;
EOF
```

### Applying migrations

Migration files live in `migrations/` at the repo root:

```bash
PGPASSWORD=<password> psql -h 127.0.0.1 -U gateway -d gateway -f migrations/001_create_api_keys.sql
PGPASSWORD=<password> psql -h 127.0.0.1 -U gateway -d gateway -f migrations/002_seed_test_keys.sql
```

### Gateway configuration

The gateway reads the database password from the `GATEWAY_DB_PASSWORD` environment variable, set in the systemd service file (`my-api-gateway.service`):

```ini
[Service]
Environment=GATEWAY_DB_PASSWORD=your_password_here
```

After changing the password, reload and restart:

```bash
sudo systemctl daemon-reload
sudo systemctl restart my-api-gateway
```

### Generating new API keys

```bash
./scripts/generate_api_key.sh
```

This outputs the raw key, prefix, hash, and an `INSERT` statement ready to run.

### Connecting manually

```bash
PGPASSWORD=<password> psql -h 127.0.0.1 -U gateway -d gateway
```

Useful queries:

```sql
-- List all keys
SELECT key_prefix, name, is_active, created_at FROM api_keys;

-- Deactivate a key
UPDATE api_keys SET is_active = false WHERE key_prefix = '27a8c227';

-- Check active connections
SELECT count(*) FROM pg_stat_activity WHERE usename = 'gateway';
```

## Service management

```bash
# Check status
sudo systemctl status my-api-gateway upstream postgresql prometheus grafana-server

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
