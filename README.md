# KubePulse

Intelligent Kubernetes health monitoring with AI-powered diagnostics and auto-remediation.

## The Problem

You're on-call at 3 AM. A pod is crashlooping. `kubectl describe` shows cryptic errors. You're piecing together logs, events, and resource metrics across multiple commands. By the time you've diagnosed the issue, you've lost 45 minutes of sleep and your SLA is at risk.

KubePulse solves this. It continuously monitors your Kubernetes cluster, detects anomalies, uses Claude AI to diagnose issues in plain English, and can automatically remediate common problems before they page you.

## Features

- **Real-time Health Monitoring**: Track pod status, node health, and cluster-wide metrics
- **AI-Powered Diagnostics**: Claude analyzes events, logs, and metrics to explain what's actually wrong
- **Auto-Remediation**: Configurable automated fixes for common issues (pod restarts, resource limits, scaling)
- **Modern Dashboard**: React/TypeScript UI with real-time updates via WebSocket
- **Alert Management**: Smart alerting that groups related issues and filters noise
- **Historical Analysis**: Track patterns and recurring issues over time

## Tech Stack

- **Backend**: Go 1.21+, Kubernetes client-go
- **Frontend**: React 18, TypeScript, TailwindCSS
- **AI**: Claude API (Anthropic)
- **Infrastructure**: Kubernetes 1.25+, Prometheus (optional)

## Quick Start

### Prerequisites

- Kubernetes cluster (local or remote)
- Valid kubeconfig with cluster access
- Anthropic API key for Claude integration

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/kubepulse.git
cd kubepulse

# Set up environment
cp .env.example .env
# Add your ANTHROPIC_API_KEY to .env

# Run with Docker Compose
docker-compose up -d

# Or build from source
make build
./bin/kubepulse --kubeconfig ~/.kube/config
```

The dashboard will be available at `http://localhost:8080`

### Configuration

Edit `config.yaml` to customize:

```yaml
cluster:
  kubeconfig: ~/.kube/config
  context: production

monitoring:
  interval: 30s
  namespaces:
    - default
    - production
    - staging

ai:
  provider: claude
  model: claude-opus-4-5-20251101
  auto_remediate: false  # Set to true to enable automatic fixes

alerts:
  slack_webhook: https://hooks.slack.com/...
  pagerduty_key: your-key-here
```

## Architecture

### Component Overview

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
│ │ Kubernetes  │ │ ← Watches cluster events
│ │   Watcher   │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │   Health    │ │ ← Analyzes metrics
│ │  Analyzer   │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │   Claude    │ │ ← AI diagnostics
│ │ Integration │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Remediation │ │ ← Applies fixes
│ │   Engine    │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         ↓
┌─────────────────┐
│ Kubernetes API  │
└─────────────────┘
```

### How It Works

1. **Watchers** continuously monitor Kubernetes resources using informers
2. **Health Analyzer** evaluates events, metrics, and patterns against known issues
3. **Claude Integration** receives context about problems and generates human-readable diagnostics
4. **Remediation Engine** (when enabled) applies fixes based on Claude's recommendations
5. **Frontend** displays real-time status and allows manual intervention

### AI Diagnostic Flow

When an issue is detected:

1. Gather context: pod logs, events, resource usage, recent deployments
2. Send to Claude with system prompt optimized for Kubernetes troubleshooting
3. Receive diagnostic with root cause, impact assessment, and recommended actions
4. Present to user via UI or execute auto-remediation if configured

## API Reference

### REST Endpoints

```
GET  /api/v1/health            - Cluster health summary
GET  /api/v1/namespaces        - List monitored namespaces
GET  /api/v1/pods              - Pod status across namespaces
GET  /api/v1/events            - Recent cluster events
GET  /api/v1/diagnostics       - AI diagnostic history
POST /api/v1/remediate         - Trigger manual remediation
```

### WebSocket

```
WS /ws - Real-time updates for health status and events
```

## Development

```bash
# Backend development
cd backend
go mod download
go run cmd/kubepulse/main.go

# Frontend development
cd frontend
npm install
npm run dev

# Run tests
make test

# Lint
make lint
```

## Deployment

### Kubernetes In-Cluster

```bash
# Deploy to cluster (runs as a Deployment)
kubectl apply -f deploy/kubernetes/

# Create secret for API key
kubectl create secret generic kubepulse-secrets \
  --from-literal=anthropic-api-key=your-key-here
```

### Helm Chart

```bash
helm install kubepulse ./charts/kubepulse \
  --set apiKey=your-anthropic-key \
  --set ingress.enabled=true \
  --set ingress.host=kubepulse.example.com
```

## Future Roadmap

- **Predictive Analytics**: Use historical data to predict failures before they happen
- **Multi-Cluster Support**: Monitor and manage multiple Kubernetes clusters from a single dashboard
- **Custom Remediation Playbooks**: Define your own automated responses to specific scenarios
- **Cost Optimization**: AI-powered recommendations for right-sizing resources based on actual usage
- **Integration Ecosystem**: Plugins for Datadog, New Relic, and other observability platforms
- **Compliance Checks**: Automated scanning for security and compliance violations (PSPs, NetworkPolicies, etc.)

## Contributing

Pull requests welcome. For major changes, open an issue first to discuss what you'd like to change.

## License

MIT
