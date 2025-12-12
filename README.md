# KubePulse

[![CI](https://github.com/charles-adedotun/kubepulse/actions/workflows/core-ci.yml/badge.svg)](https://github.com/charles-adedotun/kubepulse/actions/workflows/core-ci.yml)

Kubernetes health monitoring with AI-powered diagnostics. Watches your cluster, detects anomalies, and uses Claude to explain issues in plain English.

## Quick Eval (2 min)

### Option A: From Source (Go 1.23 + Node 20)

```bash
git clone https://github.com/charles-adedotun/kubepulse.git
cd kubepulse

# Initialize config
make config-init

# Run backend + frontend
make dev
```

- Backend API: http://localhost:8080
- Frontend dashboard: http://localhost:5173

### Option B: Docker Compose

```bash
git clone https://github.com/charles-adedotun/kubepulse.git
cd kubepulse

# Set your Anthropic API key (optional, for AI features)
export ANTHROPIC_API_KEY=your-key-here

# Build and run
docker-compose up --build
```

Dashboard available at http://localhost:8080

### What You'll See

```
┌─────────────────────────────────────────────────────────────┐
│  KubePulse Dashboard                                        │
├─────────────────────────────────────────────────────────────┤
│  Cluster: minikube    Status: Healthy    Pods: 12/12       │
│                                                             │
│  Recent Events:                                             │
│  [INFO]  pod/nginx-abc123 Running                          │
│  [WARN]  pod/api-xyz789 High memory usage (85%)            │
│  [AI]    "Memory pressure likely caused by connection      │
│          pool exhaustion. Recommend scaling replicas."     │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

- **Kubernetes cluster**: Kind, Minikube, or any K8s cluster
- **Valid kubeconfig**: `~/.kube/config` with cluster access
- **Anthropic API key**: Optional, enables AI diagnostics

## Features

- **Real-time Monitoring**: Pod, node, and service health via K8s informers
- **AI Diagnostics**: Claude analyzes events/logs and explains issues in plain English
- **Auto-Remediation**: Configurable automated fixes (disabled by default)
- **WebSocket Dashboard**: React/TypeScript UI with live updates
- **Smart Alerts**: Groups related issues, filters noise

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.23, client-go, Gorilla mux/websocket |
| Frontend | React 19, TypeScript, Vite, TailwindCSS |
| AI | Claude API (Anthropic) |
| Infra | Kubernetes 1.25+, Docker |

## Architecture

```
┌─────────────────┐
│  React Frontend │ ← WebSocket for real-time updates
└────────┬────────┘
         │
         ↓
┌─────────────────┐
│   Go Backend    │
│                 │
│ ┌─────────────┐ │
│ │ K8s Watcher │ │ ← Informers watch pods/nodes/services
│ └─────────────┘ │
│ ┌─────────────┐ │
│ │   Health    │ │ ← Runs checks every 30s
│ │  Analyzer   │ │
│ └─────────────┘ │
│ ┌─────────────┐ │
│ │   Claude    │ │ ← AI diagnostics
│ │ Integration │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ↓
┌─────────────────┐
│ Kubernetes API  │
└─────────────────┘
```

## Configuration

Copy the example config and customize:

```bash
cp .kubepulse.yaml.example ~/.kubepulse.yaml
```

Key settings in `~/.kubepulse.yaml`:

```yaml
kubernetes:
  kubeconfig: ~/.kube/config
  context: ""  # Use default context

monitoring:
  interval: 30s
  enabled_checks:
    - pod-health
    - node-health
    - service-health

ai:
  enabled: true
  timeout: 120s

server:
  port: 8080
```

## API Reference

### REST Endpoints

```
GET  /health              - Server health check
GET  /api/v1/health       - Cluster health summary
GET  /api/v1/namespaces   - List monitored namespaces
GET  /api/v1/pods         - Pod status across namespaces
GET  /api/v1/events       - Recent cluster events
GET  /api/v1/diagnostics  - AI diagnostic history
POST /api/v1/remediate    - Trigger manual remediation
```

### WebSocket

```
WS /ws - Real-time health updates and events
```

## Development

```bash
# Full dev environment setup
make setup

# Run backend only
make run

# Run frontend only (separate terminal)
make frontend-dev

# Run both (recommended)
make dev

# Run tests
make test

# Lint
make lint

# Build binary
make build
```

## Deployment

### Kubernetes In-Cluster

```bash
# Create secret for API key
kubectl create secret generic kubepulse-secrets \
  --from-literal=anthropic-api-key=your-key-here

# Deploy
kubectl apply -f deploy/kubernetes/base/
```

## CI/CD

This repo has GitHub Actions workflows for:
- **core-ci.yml**: Tests, lint, security scan, build validation
- **release.yml**: Multi-platform binary releases via GoReleaser
- **claude-review.yml**: AI-powered PR reviews

## Contributing

Pull requests welcome. For major changes, open an issue first.

## License

MIT
