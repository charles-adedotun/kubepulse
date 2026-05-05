# KubePulse

[![CI](https://github.com/charles-adedotun/kubepulse/actions/workflows/core-ci.yml/badge.svg)](https://github.com/charles-adedotun/kubepulse/actions/workflows/core-ci.yml)

KubePulse is a Go CLI and web dashboard for checking Kubernetes pod, node, and service health. It can optionally call a local Claude Code CLI for diagnostic summaries when AI features are enabled.

## What Works Today

- CLI commands: `kubepulse monitor`, `kubepulse serve`, `kubepulse check`, and `kubepulse diagnose`.
- Health checks for pods, nodes, and services using `client-go`.
- REST API and WebSocket dashboard served by the Go backend.
- React/TypeScript frontend built with Vite.
- Kubernetes manifests under `deploy/kubernetes/base`, with staging and production kustomize overlays.
- Optional AI diagnostics through the local `claude` executable. The current implementation does not call the Anthropic API directly.

## Quick Eval

### From Source

Requirements: Go 1.25.9, Node.js 20, and a kubeconfig with access to a Kubernetes cluster.

```bash
git clone https://github.com/charles-adedotun/kubepulse.git
cd kubepulse

make config-init
make dev
```

- Backend API: `http://localhost:8080`
- Frontend dev server: `http://localhost:5173`

### Docker Compose

```bash
git clone https://github.com/charles-adedotun/kubepulse.git
cd kubepulse
docker-compose up --build
```

The container mounts `${HOME}/.kube/config` read-only and serves the dashboard on `http://localhost:8080`. AI diagnostics require the Claude Code CLI to be available in the runtime environment; the Docker image does not bundle it.

## Architecture

```
React dashboard
  |
  | REST + WebSocket
  v
Go API server
  |
  +-- monitoring engine
  |     +-- pod health check
  |     +-- node health check
  |     +-- service health check
  |
  +-- optional Claude CLI diagnostics
  |
  v
Kubernetes API
```

## Configuration

Copy the example config and customize it:

```bash
cp .kubepulse.yaml.example ~/.kubepulse.yaml
```

Key settings:

```yaml
kubernetes:
  kubeconfig: ~/.kube/config
  context: ""

monitoring:
  interval: 30s
  enabled_checks:
    - pod-health
    - node-health
    - service-health

ai:
  enabled: true
  claude_path: "claude"
  timeout: 120s

server:
  port: 8080
```

Do not commit real webhook URLs, SMTP passwords, kubeconfigs, or Claude credentials. Keep secrets in local environment variables or Kubernetes Secrets.

## API Surface

Current server routes include:

```text
GET  /health
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

## Development Proof

```bash
go test ./...
cd frontend
npm ci
npm run type-check
npm run lint
npm run build
npm audit --audit-level=moderate
```

Frontend unit tests are not implemented yet; the current `npm test` script is a placeholder. CI should treat type-check, lint, build, and audit as frontend proof until real component tests are added.

## Deployment

Base manifests:

```bash
kubectl apply -f deploy/kubernetes/base/
```

Kustomize overlays:

```bash
kubectl kustomize deploy/kubernetes/staging
kubectl kustomize deploy/kubernetes/production
```

Review the default ingress hostnames and RBAC before applying to a shared cluster.

## CI/CD

- `.github/workflows/core-ci.yml`: Go tests, Go lint, frontend type/lint/build, frontend audit, Kubernetes manifest rendering, Docker build validation.
- `.github/workflows/release.yml`: GoReleaser-based tagged releases.
- `.github/workflows/claude-review.yml`: Optional Claude Code review workflow, enabled when `CLAUDE_CODE_OAUTH_TOKEN` is configured.

## License

MIT
