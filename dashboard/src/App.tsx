const agents = [
  { id: "agent-1", hostname: "ip-10-0-0-10", group: "eks", status: "active" },
  { id: "agent-2", hostname: "ip-10-0-2-23", group: "ecs", status: "active" }
];

const alerts = [
  { id: "alert-1", title: "SSH brute force", severity: "high", mitre: "T1110" },
  { id: "alert-2", title: "/etc/shadow modified", severity: "critical", mitre: "T1098" }
];

export function App() {
  return (
    <main className="shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Internal SOC</p>
          <h1>Forge SIEM</h1>
          <p className="lede">
            Single-tenant detection, endpoint telemetry, and active response for company infrastructure.
          </p>
        </div>
        <div className="stats">
          <div>
            <span>Agents</span>
            <strong>12</strong>
          </div>
          <div>
            <span>Open alerts</span>
            <strong>4</strong>
          </div>
          <div>
            <span>Events / 24h</span>
            <strong>92.8k</strong>
          </div>
        </div>
      </section>

      <section className="grid">
        <article className="panel">
          <header>
            <h2>Agents</h2>
            <a href="/opensearch-dashboards">OpenSearch Dashboards</a>
          </header>
          <ul className="rows">
            {agents.map((agent) => (
              <li key={agent.id}>
                <div>
                  <strong>{agent.hostname}</strong>
                  <span>{agent.group}</span>
                </div>
                <span className={`badge badge-${agent.status}`}>{agent.status}</span>
              </li>
            ))}
          </ul>
        </article>

        <article className="panel">
          <header>
            <h2>Alerts</h2>
            <button>Stream live</button>
          </header>
          <ul className="rows">
            {alerts.map((alert) => (
              <li key={alert.id}>
                <div>
                  <strong>{alert.title}</strong>
                  <span>{alert.mitre}</span>
                </div>
                <span className={`badge badge-${alert.severity}`}>{alert.severity}</span>
              </li>
            ))}
          </ul>
        </article>
      </section>
    </main>
  );
}
