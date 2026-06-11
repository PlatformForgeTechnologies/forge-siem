# Forge SIEM Platform Breakout Plan

This document is the execution guide for extracting the server-side code into `forge-siem-platform`.

## Goal

Create a new repository named `forge-siem-platform` containing all server-side services, shared platform libraries, dashboard, database assets, and Kubernetes deployment manifests, while leaving agent code and rules content to their own repositories.

## Scope of `forge-siem-platform`

### Keep in platform

Move these paths into `forge-siem-platform`:

- `go.mod`
- `go.sum`
- `cmd/alert-dedup/`
- `cmd/alert-indexer/`
- `cmd/api/`
- `cmd/decoder/`
- `cmd/ingest/`
- `cmd/raw-archiver/`
- `cmd/response-orchestrator/`
- `cmd/rules-engine/`
- `cmd/vuln-service/`
- `dashboard/`
- `db/`
- `app-of-apps/`
- `k8s-apps/`
- `internal/alertdedup/`
- `internal/alertindexer/`
- `internal/api/`
- `internal/archiver/`
- `internal/config/config.go`
- `internal/decoder/`
- `internal/ingest/`
- `internal/platform/`
- `internal/response/`
- `internal/rules/`
- `internal/types/event.go`
- `internal/types/message.go`
- `internal/vuln/`
- `Dockerfile`
- `Dockerfile.dashboard`
- `docs-architecture.md`
- `README.md`
- `docs/implementation-inventory-and-repo-split.md`
- `docs/platform-breakout-plan.md`

### Do not keep in platform

These belong outside the new platform repo:

- `cmd/agent/`
- `internal/agent/`
- `internal/config/agent.go`
- `agent.yaml`
- `rules/seed-rules.yaml`

Important:
- The platform still renders the agent DaemonSet and full `agent.yaml` contract from `k8s-apps/siem/templates/agent-daemonset.yaml`.
- That means the first breakout preserves a deployment-time coupling to the agent schema unless Claude also redesigns the chart ownership boundary.

## Why the pipeline stays together

These services must remain in one repo:

- ingest
- decoder
- raw archiver
- rules engine
- alert indexer
- alert dedup
- response orchestrator

They are tightly coupled by:

- JSON envelope/event schema
- Redis stream names
- consumer group contracts
- dedup key behavior
- alert document shape

Independent repo versioning here would create immediate schema drift risk.

## Suggested target tree

```text
forge-siem-platform/
  cmd/
    alert-dedup/
    alert-indexer/
    api/
    decoder/
    ingest/
    raw-archiver/
    response-orchestrator/
    rules-engine/
    vuln-service/
  dashboard/
  db/
  docs/
  internal/
    alertdedup/
    alertindexer/
    api/
    archiver/
    config/
    decoder/
    ingest/
    platform/
    response/
    rules/
    types/
    vuln/
  k8s-apps/
  app-of-apps/
  Dockerfile
  Dockerfile.dashboard
  go.mod
  go.sum
  README.md
```

## Module and import changes

### Immediate option

Keep the module path as `forge-siem` temporarily during extraction to minimize churn, then rename once the repo is stable.

Pros:
- lowest-risk move
- fewer simultaneous changes

Cons:
- temporary mismatch between repo name and module path

### Preferred final state

Rename module path from:

- `module forge-siem`

to:

- `module forge-siem-platform`

Then update imports:

- `forge-siem/internal/...` -> `forge-siem-platform/internal/...`

Files affected:

- all `cmd/*/main.go`
- all `internal/*/*.go`

## File move map

### Command entrypoints

- `cmd/alert-dedup/main.go`
- `cmd/alert-indexer/main.go`
- `cmd/api/main.go`
- `cmd/decoder/main.go`
- `cmd/ingest/main.go`
- `cmd/raw-archiver/main.go`
- `cmd/response-orchestrator/main.go`
- `cmd/rules-engine/main.go`
- `cmd/vuln-service/main.go`

### Shared platform code

- `internal/alertdedup/service.go`
- `internal/alertindexer/service.go`
- `internal/api/server.go`
- `internal/archiver/service.go`
- `internal/config/config.go`
- `internal/decoder/service.go`
- `internal/ingest/service.go`
- `internal/platform/metrics.go`
- `internal/platform/naming.go`
- `internal/platform/opensearch.go`
- `internal/platform/redis.go`
- `internal/platform/runtime.go`
- `internal/platform/stream_worker.go`
- `internal/response/service.go`
- `internal/rules/service.go`
- `internal/types/event.go`
- `internal/types/message.go`
- `internal/vuln/service.go`

### Platform assets

- `dashboard/index.html`
- `dashboard/nginx.conf`
- `dashboard/package.json`
- `dashboard/src/App.tsx`
- `dashboard/src/main.tsx`
- `dashboard/src/styles.css`
- `dashboard/tsconfig.json`
- `dashboard/vite.config.ts`
- `db/schema.sql`
- `k8s-apps/siem/Chart.yaml`
- `k8s-apps/siem/values.yaml`
- `k8s-apps/siem/templates/*`
- `app-of-apps/k8s-apps/siem.yaml`

## Readme suggestion

The new `forge-siem-platform` README should say clearly:

- this repo owns the server-side SIEM platform
- the agent lives separately
- rules content lives separately
- the platform currently consumes agent JSON envelopes over mTLS

## Extraction sequence

### Step 1

Create the new `forge-siem-platform` repo and copy the platform-scoped files listed above.

### Step 2

Run:

- `go test ./...`
- `helm template forge-siem k8s-apps/siem`

Fix import/module issues before any cleanup.

### Step 3

Update module path if desired from `forge-siem` to `forge-siem-platform`.

### Step 4

Patch documentation references so they no longer mention agent/rules content as local paths inside the same repo.

### Step 5

Only after validation succeeds, remove moved platform files from the original mixed repo or let Claude complete the full split.

## Compatibility assumptions

The platform currently assumes:

- agent transport is newline-delimited JSON envelopes over mTLS
- Redis stream names:
  - `siem:raw-events`
  - `siem:decoded-events`
  - `siem:alerts`
  - `siem:deduped-alerts`
  - `siem:response-acks`
- OpenSearch index families:
  - `siem-raw-*`
  - `siem-events-*`
  - `siem-alerts-*`

These must remain stable during the initial split.

## Migration risks

### High risk

- Moving `internal/types/event.go` and `internal/types/message.go` without preserving exact schemas will break all pipeline services.
- Duplicating those protocol structs across agent and platform repos without a versioned contract or contract-test coverage creates silent runtime drift.
- Renaming the module path and moving files in the same step increases failure surface.
- Accidentally moving `internal/config/agent.go` into platform will create an unnecessary config dependency on the agent repo.
- Extracting `forge-siem-rules` before platform can consume external rule packs creates a repo split without runtime decoupling.

### Medium risk

- Dashboard/API proxy assumptions may break if repo-local container builds are not carried over.
- Helm chart references may drift if image names are changed during extraction.
- The docs currently describe pending features that span repo boundaries; they need updates after the split lands.

## Validation checklist

- `go test ./...` passes in the new platform repo
- `helm template forge-siem k8s-apps/siem` passes
- `cmd/rules-engine`, `cmd/alert-indexer`, and `cmd/alert-dedup` all still build
- the chart still renders:
  - `siem-alert-indexer`
  - `siem-alert-dedup`
  - `siem-rules-engine`
  - `siem-ingest`
  - `siem-decoder`

## What Claude should not change during breakout

- Do not redesign stream names
- Do not redesign event or alert schemas
- Do not convert the wire format to Protobuf during the breakout
- Do not split pipeline services into separate repos
