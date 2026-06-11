# Implementation Inventory And Repo Split

This document records what is currently implemented in `forge-siem`, what remains pending, and how to split the codebase into three repositories without breaking the current architecture.

## Current implementation

### Implemented today

#### Agent

Files:
- `cmd/agent/main.go`
- `internal/agent/service.go`
- `internal/config/agent.go`
- `agent.yaml`

Implemented:
- Agent config loading from YAML
- mTLS client connection to ingest
- Heartbeat emission
- Log collection by polling configured file paths
- Support for host log paths including:
  - `/var/log/auth.log`
  - `/var/log/syslog`
  - `/var/log/messages`
  - `/var/log/containers/*.log`
  - `/var/log/pods/*/*/*.log`
  - `/var/lib/docker/containers/*/*.log`
- JSON envelope transport to ingest
- Reconnect handling when transport fails
- Offset advancement only after successful send
- Local file offsets persisted to disk across restarts
- Basic FIM path exclusion validation for `/proc`, `/sys`, `/dev`, `/run`
- Optional Loki output feature flags and config wiring in the chart
- Agent config surface prepared for both:
  - OSS Loki: `push_url` only
  - Enterprise / multi-tenant Loki: `push_url` plus optional `tenant_id`, `username`, and `password`
- Loki push client implemented with path-based fanout control

Not implemented yet:
- Protobuf transport
- inotify-based tailing
- SQLite-backed FIM state
- process monitoring
- active response command execution
- certificate rotation without reconnect
- agent identity binding to certificate subject/SAN

#### Server pipeline

Files:
- `cmd/ingest/main.go`
- `cmd/decoder/main.go`
- `cmd/raw-archiver/main.go`
- `cmd/alert-indexer/main.go`
- `cmd/alert-dedup/main.go`
- `cmd/rules-engine/main.go`
- `cmd/response-orchestrator/main.go`
- `cmd/vuln-service/main.go`
- `internal/ingest/service.go`
- `internal/decoder/service.go`
- `internal/archiver/service.go`
- `internal/alertindexer/service.go`
- `internal/alertdedup/service.go`
- `internal/rules/service.go`
- `internal/response/service.go`
- `internal/vuln/service.go`
- `internal/types/*.go`
- `internal/platform/*.go`

Implemented:
- Ingest listener on TCP 1514 with mTLS server auth
- Newline-delimited JSON envelope admission
- Raw event writes into Redis Streams
- Heartbeat TTL writes to Redis
- Decoder consumer group on raw stream
- Raw archiver consumer group on raw stream
- Shared Redis Streams consumer helper with `XACK`
- Pending-entry recovery using `XAUTOCLAIM`
- Basic event normalization into a common schema
- Decoded event writes into Redis Streams
- Raw event indexing into daily `siem-raw-*`
- Decoded event indexing into daily `siem-events-*`
- Seed rules engine structure with simple match logic
- Rules engine consumer group on decoded stream
- Alert writes into `siem:alerts`
- Alert indexer consumer group on alerts stream
- Alert indexing into daily `siem-alerts-*`
- Alert dedup worker on alerts stream with 60-second Redis TTL suppression
- Response orchestrator scaffold
- Vulnerability service scaffold
- OpenSearch client with optional basic auth
- Redis client wrapper
- Metrics server with `/healthz` and `/metrics`
- Prometheus stream lag gauges for KEDA/autoscaling queries

Not implemented yet:
- Protobuf decoding path
- PostgreSQL repositories and persistence layer
- compliance tagging
- dedup stream worker
- RabbitMQ exchanges, DLQ, retries, routing keys
- response command delivery and ACK flow
- real Sigma parser/evaluator
- real correlation storage windows
- NVD ingestion and package matching logic
- CloudTrail collector
- OpenSearch index templates / ILM / retention bootstrap

#### API and dashboard

Files:
- `cmd/api/main.go`
- `internal/api/server.go`
- `dashboard/*`

Implemented:
- Fiber API scaffold
- Read and write endpoint shapes for agents, alerts, rules, vulnerability, stats
- Bearer token guard for API and websocket route
- API server timeouts and body limit
- Basic React dashboard shell
- Dashboard containerization via Nginx
- Dashboard reverse proxy to API paths

Not implemented yet:
- real database-backed API handlers
- websocket alert stream
- OIDC / SSO / RBAC
- embedded OpenSearch Dashboards integration
- MITRE heatmap
- vulnerability view
- agent detail workflows

#### Deployment and operations

Files:
- `k8s-apps/siem/*`
- `app-of-apps/k8s-apps/siem.yaml`
- `Dockerfile`
- `Dockerfile.dashboard`

Implemented:
- Helm chart for the platform
- ArgoCD application entry
- Agent DaemonSet
- Deployments for ingest, decoder, raw archiver, rules engine, response orchestrator, vuln service, API, dashboard
- OpenSearch StatefulSet
- RabbitMQ Deployment
- Traefik IngressRoute for dashboard
- Traefik IP allowlist middleware
- ServiceMonitor for metrics
- KEDA ScaledObjects based on Prometheus queries
- Multi-stage Go container build
- Dashboard production image build
- ExternalSecrets-based secret integration
- Secret split:
  - `siem-runtime-secrets`
  - `siem-agent-tls`
  - `siem-opensearch-secrets`

Not implemented yet:
- production OpenSearch multi-node bootstrap logic
- OpenSearch TLS/cert wiring inside the chart
- strict per-service Kubernetes service accounts / RBAC
- full secret separation per workload
- CI/CD pipelines for image publishing, tests, Helm lint, and release automation

## Current validation state

Validated locally in this repo:
- `go test ./...`
- `helm template forge-siem k8s-apps/siem`

This means the repo currently has a coherent build/render state, but not a production-complete runtime state.

## Pending work

### High priority functional gaps

1. Replace JSON envelopes with Protobuf on the agent-ingest wire protocol.
2. Persist remaining agent local state:
   - FIM baseline
   - process baseline
3. Implement PostgreSQL persistence for:
   - agents
   - rules
   - alerts
   - response logs
   - CVE data
4. Implement remaining alert pipeline workers:
   - compliance tagger
5. Implement RabbitMQ topology and active response flow end to end.
6. Implement real Sigma parsing and matching instead of placeholder string matching.
7. Implement NVD ingestion and package enumeration/matching.

### High priority security and production gaps

1. Bind agent identity to the mTLS certificate identity.
2. Replace bearer-token API auth with OIDC/RBAC.
3. Finish OpenSearch production hardening:
   - TLS
   - auth
   - index templates
   - retention / lifecycle
   - multi-node formation for prod
4. Reduce agent host privileges further where possible.
5. Split secrets more narrowly if the runtime grows more sensitive.

### Medium priority platform gaps

1. Improve stream lag metrics from approximate `XLEN` to consumer-group aware lag.
2. Add real `/metrics` instrumentation for service-specific counters and errors.
3. Add tests beyond compile checks.
4. Add dashboards and alerts for platform health.

## Repo split decision

The codebase should now be split into three repositories.

### Repo 1: `forge-siem-agent`

Scope:
- the host/node agent only

Move:
- `cmd/agent/`
- `internal/agent/`
- `internal/config/agent.go`
- `internal/config/config.go`
- `internal/platform/runtime.go`
- agent-side message and event contracts that are required by the agent:
  - `internal/types/event.go`
  - `internal/types/message.go`
- `agent.yaml`
- agent-specific build and release assets
- `go.mod`
- `go.sum`

Why:
- independent host release cadence
- cleaner binary distribution story
- likely open-source/distribution candidate
- different contributor audience than the platform

Notes:
- The current agent code is already largely isolated.
- The current agent entrypoint still depends on `internal/config/config.go` and `internal/platform/runtime.go`; extraction must either move those files or replace them with agent-owned equivalents before the repo can build.
- Shared contracts should not be treated as carefree copies. The wire schema needs an explicit versioned source of truth or contract-test coverage to avoid drift between agent and platform.

### Repo 2: `forge-siem-platform`

Scope:
- all server-side services and deployment assets

Move:
- `go.mod`
- `go.sum`
- `cmd/alert-dedup/`
- `cmd/alert-indexer/`
- `cmd/ingest/`
- `cmd/decoder/`
- `cmd/raw-archiver/`
- `cmd/rules-engine/`
- `cmd/response-orchestrator/`
- `cmd/vuln-service/`
- `cmd/api/`
- `internal/alertdedup/`
- `internal/alertindexer/`
- `internal/ingest/`
- `internal/decoder/`
- `internal/archiver/`
- `internal/rules/`
- `internal/response/`
- `internal/vuln/`
- `internal/api/`
- `internal/platform/`
- shared runtime config:
  - `internal/config/config.go`
- server-side shared types:
  - `internal/types/event.go`
  - `internal/types/message.go`
- `dashboard/`
- `db/`
- `k8s-apps/`
- `app-of-apps/`
- `Dockerfile`
- `Dockerfile.dashboard`
- deployment documentation
- `README.md`
- `docs-architecture.md`
- `docs/implementation-inventory-and-repo-split.md`
- `docs/platform-breakout-plan.md`

Why:
- shared infrastructure dependencies
- same Kubernetes deployment lifecycle
- same Helm chart and secret model
- strong coupling through Redis stream contracts

Notes:
- This repo should remain a monorepo for the platform services for now.
- Do not split ingest/decoder/archiver/rules/orchestrator into separate repos.
- The platform chart still owns the agent image reference and the rendered `agent.yaml` schema today. Independent agent versioning therefore requires an explicit compatibility/versioning plan at that chart boundary.

### Repo 3: `forge-siem-rules`

Scope:
- detection content only

Move:
- `rules/seed-rules.yaml`
- future Sigma packs
- future community detection content
- rule metadata and release notes

Why:
- independent content release cadence
- easier community contribution
- engine updates should not be required to ship new rules
- matches the industry pattern used by Sigma and similar detection ecosystems

Notes:
- The platform should eventually consume this repo as versioned content, not as source code copied by hand.
- Short term, a simple release artifact or git submodule/subtree can work.
- The current running platform does not yet load external rule packs. Extracting rules first is repository preparation, not the completion of runtime decoupling.

## What should not be split

Do not split these into independent repos:
- ingest
- decoder
- raw archiver
- rules engine
- response orchestrator

Reason:
- they share the same event contracts
- they are coupled by Redis stream schemas and consumer-group behavior
- a schema change in one service usually requires coordinated changes in the others

Splitting them now would add:
- CI complexity
- version skew risk
- rollout coordination overhead

without adding a real product or operational benefit.

## Suggested extraction order

1. Extract `forge-siem-rules` first.
   - lowest-risk split
   - almost no code dependency
   - but it does not create runtime rule decoupling until platform consumption is implemented

2. Extract `forge-siem-agent` second.
   - already self-contained enough
   - easiest independent build/release story

3. Rename the current remaining repo to `forge-siem-platform`.
   - least disruption
   - preserves backend service cohesion

See also:
- `docs/platform-breakout-plan.md`

## Suggested migration checklist

### Phase 1

- Create `forge-siem-rules`
- Move `rules/seed-rules.yaml`
- Replace local file reference in platform with a documented external source assumption

### Phase 2

- Create `forge-siem-agent`
- Move agent code and config
- Replace or move the pieces the agent still compiles against:
  - `internal/config/config.go`
  - `internal/platform/runtime.go`
- Add explicit wire-contract ownership or contract tests for envelope/event compatibility
- Add agent-specific Docker/build/release pipeline

### Phase 3

- Rename this repo to `forge-siem-platform`
- Remove moved agent/rules files
- Update Helm/chart/docs references
- Update import paths and module names

## Immediate follow-up after split

After the split, the first engineering tasks should be:

1. Introduce a versioned schema contract for the agent-platform wire format.
2. Replace JSON with Protobuf on that contract.
3. Define how the platform consumes rule packs from `forge-siem-rules`.
4. Add release/version compatibility documentation between:
   - agent
   - platform
   - rules
