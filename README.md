# Forge SIEM

Single-tenant SIEM and EDR platform for Kubernetes. Ingests logs from enrolled agents and agentless sources, detects threats using Sigma-compatible rules, and surfaces alerts in a real-time dashboard.

Designed for teams that want Wazuh-level coverage without the operational weight — cloud-native, self-contained, and deployable on any Kubernetes cluster.

---

## Repositories

| Repo | Purpose |
|---|---|
| **forge-siem** (this repo) | Umbrella Helm chart — install everything from here |
| [forge-siem-platform](../forge-siem-platform) | Go services, API, dashboard, platform Helm chart |
| [forge-siem-agent](../forge-siem-agent) | Host agent — FIM, process monitoring, active response |
| [forge-siem-rules](../forge-siem-rules) | Sigma detection rule packs |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Cluster                                                    │
│                                                             │
│  ┌─────────────────┐   ┌───────────────────────────────┐   │
│  │  Agent DaemonSet│   │  Cluster Logs DaemonSet       │   │
│  │  (per node)     │   │  tails /var/log/pods/*        │   │
│  │  FIM · procs    │   │  zero config, all pod logs    │   │
│  │  inventory      │   └──────────────┬────────────────┘   │
│  └────────┬────────┘                  │                     │
│           │ mTLS 1514                 │ Redis stream        │
│  ┌────────▼───────────────────────────▼────────────────┐   │
│  │  Ingest / Collector → siem:raw-events               │   │
│  │  Decoder · Rules Engine · Alert pipeline            │   │
│  │  PostgreSQL · OpenSearch · RabbitMQ                 │   │
│  └──────────────────────────┬──────────────────────────┘   │
│                             │                               │
│  ┌──────────────────────────▼──────────────────────────┐   │
│  │  Dashboard  ←  API (REST + SSE)                     │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘

External agents (other clusters / bare metal) connect via
siem-ingest-external LoadBalancer when enabled.
```

---

## Quick install

### Prerequisites

- Kubernetes cluster (any distribution)
- Helm 3.14+
- PostgreSQL 15+ (BYO)
- Redis 7+ (BYO)
- OpenSearch and RabbitMQ are deployed in-cluster by default

### 1. Create required secrets

```bash
kubectl create namespace siem

kubectl create secret generic siem-runtime-secrets -n siem \
  --from-literal=postgres_dsn='postgres://user:pass@host:5432/siem?sslmode=require' \
  --from-literal=redis_password='your-redis-password' \
  --from-literal=api_auth_token='$(openssl rand -hex 32)' \
  --from-literal=response_hmac_key='$(openssl rand -hex 32)'

# Generate a CA for agent TLS (or bring your own)
openssl req -x509 -newkey rsa:4096 -keyout ca.key -out ca.crt -days 3650 -nodes \
  -subj "/CN=forge-siem-ca"

kubectl create secret generic siem-agent-tls -n siem \
  --from-file=ca.crt=ca.crt \
  --from-file=ca.key=ca.key

kubectl create secret generic siem-opensearch-secrets -n siem \
  --from-literal=opensearch_username=admin \
  --from-literal=opensearch_password='$(openssl rand -hex 16)' \
  --from-literal=opensearch_initial_admin_password='$(openssl rand -hex 16)'
```

### 2. Install

**From GHCR OCI (v0.2.0+):**
```bash
helm install forge-siem \
  oci://ghcr.io/platformforgetechnologies/forge-siem/charts/forge-siem \
  --version 0.2.0 \
  --namespace siem \
  -f your-values.yaml
```

**Local (both repos checked out side by side):**
```bash
./scripts/bundle-charts.sh
helm install forge-siem k8s-apps/forge-siem --namespace siem
```

### 3. Verify

```bash
helm test forge-siem -n siem
```

The smoke test checks API health, dashboard health, OpenSearch index templates, and runs an enrollment flow.

---

## Configuration

Override defaults in a `values.yaml` file passed with `-f`.

### Dependency modes

```yaml
# In platform values:
dependencies:
  postgres:
    mode: byo           # always BYO — point at your existing instance
  redis:
    mode: byo           # always BYO
  opensearch:
    mode: self_hosted   # or: byo
  rabbitmq:
    mode: self_hosted   # or: byo
```

### External agent access

By default, ingest is `ClusterIP` — agents on the same cluster connect automatically via internal DNS. To support agents outside the cluster:

```yaml
platform:
  ingest:
    externalService:
      enabled: true
      type: LoadBalancer
      annotations:
        # AWS:   service.beta.kubernetes.io/aws-load-balancer-type: nlb
        # GCP:   cloud.google.com/load-balancer-type: Internal
        # Azure: service.beta.kubernetes.io/azure-load-balancer-internal: "true"
  api:
    enrollment:
      ingestHost: <your-external-lb-hostname>
```

### Disable agent

```yaml
agent:
  enabled: false
```

### Disable cluster-log DaemonSet

```yaml
platform:
  clusterLogs:
    enabled: false
```

---

## Upgrade

```bash
helm upgrade forge-siem k8s-apps/forge-siem --namespace siem
# or from OCI:
helm upgrade forge-siem oci://.../forge-siem --version 0.3.0 --namespace siem
```

---

## Releases

| Version | Highlights |
|---|---|
| v0.2.0 | Collector service, zero-config cluster log collection, agentless network ingest, umbrella chart, vendor-agnostic ingest service |
| v0.1.1 | Bulk OpenSearch indexing, Logs + Network pages, Suricata/Zeek/VPC/Traefik parsers |
| v0.1.0 | Initial release — all core services, dashboard, enrollment, active response |
