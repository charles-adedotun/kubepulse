# KubePulse

[![CI](https://github.com/charles-adedotun/kubepulse/actions/workflows/core-ci.yml/badge.svg)](https://github.com/charles-adedotun/kubepulse/actions/workflows/core-ci.yml)

KubePulse is a Kubernetes health and observability tool for quickly checking cluster status from a Go CLI or a web dashboard. It runs pod, node, and service health checks against the Kubernetes API, aggregates results into a traffic-light cluster view, and exposes the same data through REST and WebSocket endpoints.

The repository also includes experimental AI-assisted diagnostics, predictive insights, and remediation helpers. Those paths depend on a local `claude` executable and should be treated as development-stage features rather than production automation.

## What It Does

- Runs Kubernetes health checks from the CLI with `kubepulse monitor`, `kubepulse check`, and `kubepulse diagnose`.
- Serves a Go API and React dashboard with live health updates over WebSocket.
- Checks pod phases, readiness, pending failure reasons, and restart counts.
- Checks node readiness and pressure conditions, with placeholder node resource usage until metrics API integration is implemented.
- Checks services for ready endpoints.
- Tracks health results, emits metrics from checks, and evaluates default alert rules through the in-process engine.
- Provides Kubernetes deployment manifests under `deploy/kubernetes/base` with staging and production kustomize overlays.

## Quickstart

Requirements:

- Go 1.25.10
- Node.js 20
- A kubeconfig with access to the cluster you want to inspect

Clone and run from source:

```bash
git clone https://github.com/charles-adedotun/kubepulse.git
cd kubepulse

make config-init
make dev
```

Development servers:

- Backend API and bundled dashboard: `http://localhost:8080`
- Frontend Vite server: `http://localhost:5173`

Build or install the CLI:

```bash
make build
./bin/kubepulse --help

make install
kubepulse --help
```

Run with Docker Compose:

```bash
docker-compose up --build
```

The compose setup mounts `${HOME}/.kube/config` read-only and serves KubePulse on `http://localhost:8080`.

## Common Commands

```bash
# Run the default monitor once
kubepulse monitor

# Continuously monitor pods and nodes
kubepulse monitor --watch --interval 30s

# Limit checks to one namespace
kubepulse monitor --namespace default

# Run an individual check
kubepulse check pod-health
kubepulse check node-health

# Start the dashboard and API
kubepulse serve --port 8080

# Run AI-assisted diagnostics for an unhealthy check
kubepulse diagnose pod-health
```

Use `--kubeconfig` and `--context` to override the default kubeconfig selection.

## Configuration

Create a local config file from the example:

```bash
cp .kubepulse.yaml.example ~/.kubepulse.yaml
```

Important settings include:

```yaml
kubernetes:
  kubeconfig: ~/.kube/config
  context: ""
  namespaces: []

monitoring:
  interval: 30s
  enabled_checks:
    - pod-health
    - node-health
    - service-health

server:
  port: 8080
  enable_web: true

ui:
  refresh_interval: 10s
```

Keep webhook URLs, SMTP credentials, kubeconfigs, and Claude credentials out of commits. Use local environment variables or Kubernetes Secrets for sensitive values.

## Checks And Signals

| Check | What it inspects | Current notes |
| --- | --- | --- |
| `pod-health` | Pod phase, readiness, pending error reasons, restart counts, namespace exclusions | Defaults exclude `kube-system` and `kube-public`. |
| `node-health` | Node readiness plus memory, disk, and PID pressure conditions | CPU and memory usage are currently placeholder values, not metrics API readings. |
| `service-health` | Service endpoints with ready addresses | Services with no ready endpoints are marked degraded. |

The monitor engine runs registered checks on an interval, stores the latest result for each check, evaluates alert rules, runs anomaly detection over emitted metrics, and can start AI analysis for degraded or unhealthy checks when AI is enabled.

## Architecture

```text
React dashboard
  |
  | REST API and WebSocket
  v
Go API server
  |
  +-- monitoring engine
  |     +-- pod-health
  |     +-- node-health
  |     +-- service-health
  |
  +-- alerts, metrics, SLO, and anomaly helpers
  |
  +-- optional Claude CLI diagnostics
  |
  v
Kubernetes API
```

Primary API routes include:

```text
GET  /api/v1/health
GET  /api/v1/health/cluster
GET  /api/v1/health/checks
GET  /api/v1/health/checks/{name}
GET  /api/v1/alerts
GET  /api/v1/metrics
GET  /api/v1/config/ui
GET  /api/v1/contexts
GET  /api/v1/contexts/current
POST /api/v1/contexts/switch
GET  /api/v1/ai/insights
POST /api/v1/ai/analyze/{check}
POST /api/v1/ai/heal/{check}
POST /api/v1/ai/assistant/query
GET  /api/v1/ai/predictions
GET  /api/v1/ai/remediation/{check}/suggestions
POST /api/v1/ai/remediation/execute
GET  /api/v1/ai/alerts/insights
WS   /ws
```

## Testing And CI

Local checks:

```bash
go test ./...
go vet ./...

cd frontend
npm ci
npm run type-check
npm run lint
npm run build
npm audit --audit-level=moderate
```

The GitHub Actions setup runs backend tests with race detection and coverage, frontend type/lint/build checks, dependency and security scans, Kubernetes manifest rendering, and binary/Docker build validation.

See [docs/github-workflows.md](docs/github-workflows.md) for the workflow details.

## Deployment Manifests

Render or apply the Kubernetes manifests before customizing them for a shared cluster:

```bash
kubectl kustomize deploy/kubernetes/base
kubectl kustomize deploy/kubernetes/staging
kubectl kustomize deploy/kubernetes/production
```

Review the generated RBAC, service account, image, hostnames, and namespace strategy before applying.

## Status And Limitations

KubePulse is actively evolving and should be evaluated before production use.

- Frontend `npm test` is a placeholder; current frontend proof is type-check, lint, build, and audit.
- `kubepulse monitor --output json` and `--output yaml` are declared but not implemented yet.
- Node resource usage does not currently query the Kubernetes metrics API.
- The alerts API currently returns example alert data rather than persisted alert history.
- AI diagnostics require a local Claude Code CLI executable; the main app does not bundle Claude.
- The Docker image does not include a kubeconfig or Claude CLI.

## License

MIT
