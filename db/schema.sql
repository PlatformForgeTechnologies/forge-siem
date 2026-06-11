CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE agents (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  hostname        TEXT NOT NULL,
  ip              INET,
  os              TEXT,
  agent_version   TEXT,
  status          TEXT DEFAULT 'active',
  last_seen       TIMESTAMPTZ,
  enrolled_at     TIMESTAMPTZ DEFAULT NOW(),
  groups          TEXT[] DEFAULT '{}'
);

CREATE TABLE rules (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title           TEXT NOT NULL,
  sigma_yaml      TEXT NOT NULL,
  enabled         BOOLEAN DEFAULT true,
  severity        TEXT,
  mitre_tags      TEXT[],
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE alerts (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id         UUID REFERENCES rules(id),
  agent_id        UUID REFERENCES agents(id),
  severity        TEXT,
  title           TEXT,
  os_id           TEXT,
  status          TEXT DEFAULT 'open',
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  resolved_at     TIMESTAMPTZ
);

CREATE TABLE cves (
  id              TEXT PRIMARY KEY,
  description     TEXT,
  cvss_score      FLOAT,
  severity        TEXT,
  published       TIMESTAMPTZ
);

CREATE TABLE cve_affected (
  cve_id          TEXT REFERENCES cves(id),
  product         TEXT,
  version_start   TEXT,
  version_end     TEXT,
  fix_version     TEXT
);

CREATE TABLE response_log (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  alert_id        UUID REFERENCES alerts(id),
  agent_id        UUID REFERENCES agents(id),
  action          TEXT,
  params          JSONB,
  status          TEXT,
  executed_at     TIMESTAMPTZ,
  result          JSONB
);

CREATE TABLE alert_compliance (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  alert_id        UUID REFERENCES alerts(id),
  framework       TEXT NOT NULL,
  control_id      TEXT NOT NULL,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);
