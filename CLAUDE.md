# forge-siem — LEGACY MONOREPO

> **This repo is the original development monorepo. It is no longer the source of truth.**
> Active development now happens in three separate repos:
> - **forge-siem-agent** — standalone host agent
> - **forge-siem-platform** — server-side services, API, dashboard, Helm chart
> - **forge-siem-rules** — Sigma detection content
>
> Do not start new features here. Use this repo only to review history or as a reference
> when the split repos need context about original design decisions.

## What This Was
Single-tenant SIEM + EDR platform. Go backend, React dashboard, Kubernetes-native.
Target: K8s-heavy enterprise environments that find Wazuh operationally too heavy.

## Tech Stack
- **Backend**: Go 1.23, Fiber v2, Redis Streams, OpenSearch, PostgreSQL, RabbitMQ
- **Dashboard**: React 18, TypeScript, Vite → served by Nginx
- **Infra**: Helm, ExternalSecrets (AWS Secrets Manager), Traefik ingress, Prometheus metrics
- **Module**: `module forge-siem` (single monorepo, splitting into 3 repos — see below)

## Service Map

| Service | Entry | Status |
|---|---|---|
| agent | `cmd/agent` | Production-ready |
| ingest | `cmd/ingest` | Production-ready |
| decoder | `cmd/decoder` | Production-ready |
| raw-archiver | `cmd/raw-archiver` | Production-ready |
| rules-engine | `cmd/rules-engine` | Scaffolded — fake ticker, not consuming events |
| response-orchestrator | `cmd/response-orchestrator` | Scaffolded — no RabbitMQ impl |
| vuln-service | `cmd/vuln-service` | Scaffolded — no NVD feed parsing |
| api | `cmd/api` | Mock data only — PostgreSQL unused |

## Data Flow
```
Agent → mTLS/TLS1.3 → Ingest → siem:raw-events (Redis)
  ├→ Raw Archiver → OpenSearch siem-raw-YYYY.MM.DD
  └→ Decoder → OpenSearch siem-events-YYYY.MM.DD
                   └→ siem:decoded-events (Redis)
                         └→ Rules Engine → siem:alerts (NOT YET WRITTEN)
                                             └→ Response Orchestrator → RabbitMQ (NOT YET IMPL)
```

## Shared Internal Packages
- `internal/types` — wire format: `Envelope`, `AgentHeartbeat` (shared by agent + ingest)
- `internal/config` — env-var config loader; `AppConfig` + `AgentFileConfig`
- `internal/platform` — Redis client, OpenSearch client, StreamWorker, Prometheus metrics, graceful Run()

## Critical Gaps (in priority order)
1. Rules Engine: must consume `siem:decoded-events`, parse `rules/seed-rules.yaml` (Sigma), write to `siem:alerts`
2. Alert persistence: API must write/read alerts from PostgreSQL (`db/schema.sql` is defined, unused)
3. Response Orchestrator: implement RabbitMQ topology; agent needs command receiver
4. Alert dedup: SHA256 key generated but never checked against Redis
5. Dashboard: wire to REST API (all data is currently hardcoded)
6. WebSocket `/ws/alerts`: returns 501
7. Vuln Service: NVD feed download + package matching

## DB Schema (defined, not yet used)
Tables: `agents`, `rules`, `alerts`, `cves`, `cve_affected`, `response_log`, `alert_compliance`
Location: `db/schema.sql`

## Planned Repo Split
This monorepo is being split into three repos. **Do not move or delete files until split is confirmed complete.**

| Repo | Contents |
|---|---|
| `forge-siem-agent` | `cmd/agent`, `internal/agent`, `internal/config`, `internal/types`, subset of `internal/platform` |
| `forge-siem-platform` | All other services, dashboard, db, k8s-apps |
| `forge-siem-rules` | `rules/` — Sigma YAML detection content only |

## Conventions
- Each service exports a single `Service` struct with a `Run(ctx context.Context) error` method
- `platform.Run()` wraps every service for graceful shutdown
- Metrics server always on port 9090; main service on its own port
- All secrets via env vars; `config.Load(serviceName)` at startup
- Stream consumer groups use `XAutoClaim` for pending message recovery
- OpenSearch indices are date-rotated: `siem-<type>-YYYY.MM.DD`
- No ORM — raw SQL via `database/sql`
