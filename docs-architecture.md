# Architecture Notes

## Runtime boundaries

- `agent`: node and host telemetry, plus active response execution
- `ingest`: mTLS termination and raw event admission
- `decoder`: normalization into the shared event schema
- `rules-engine`: Sigma-compatible evaluation and correlation
- `vuln-service`: CVE feed ingestion and package matching
- `response-orchestrator`: RabbitMQ-backed active response routing
- `api`: operator-facing control plane for agents, alerts, rules, and metrics
- `dashboard`: React UI and embedded OpenSearch search workflows

## Reused infrastructure

- Aurora PostgreSQL: add database `siem`
- ElastiCache Redis: use logical DB `2`
- EKS cluster: existing `PlatformForgeTechnologies` production cluster
- Secrets: ExternalSecrets + AWS Secrets Manager
- Ingress: Traefik + internal NLB
- Monitoring: Prometheus/Grafana scrape service metrics

## Container logs

- Kubernetes pod/container logs should be collected at node level.
- The current agent path supports `/var/log/containers/*.log`, `/var/log/pods/*/*/*.log`, and Docker container JSON logs.
- Recommended production model:
  - OpenSearch for normalized security telemetry and alert investigations
  - Loki for bulk container/application logs and operational debugging
  - Promote selected failure patterns from Loki or the agent stream into the SIEM alert pipeline when they become security-relevant
- Loki should remain feature-flagged and additive, with SIEM output still enabled by default.

## Gaps still to implement

- Real Redis Streams producers and consumer groups with `XREADGROUP`, `XACK`, and pending recovery
- PostgreSQL repositories and migrations runner
- OpenSearch bulk indexing and lifecycle policies
- mTLS agent enrollment and certificate rotation plumbing
- RabbitMQ exchange, queues, DLQ, and retry topology
- FIM SQLite local state, process baseline diffing, and command execution safeguards

## Secret layout

- `siem-runtime-secrets`: Redis, PostgreSQL, RabbitMQ, API auth, response HMAC, optional Loki credentials
- `siem-agent-tls`: agent and ingest TLS material (`ca.crt`, `tls.crt`, `tls.key`)
- `siem-opensearch-secrets`: OpenSearch URL, service credentials, and initial admin bootstrap password
