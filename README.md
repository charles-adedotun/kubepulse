# KubePulse

[![Go CI](https://github.com/charles-adedotun/kubepulse/actions/workflows/go-ci.yml/badge.svg)](https://github.com/charles-adedotun/kubepulse/actions/workflows/go-ci.yml)
[![Frontend CI](https://github.com/charles-adedotun/kubepulse/actions/workflows/frontend-ci.yml/badge.svg)](https://github.com/charles-adedotun/kubepulse/actions/workflows/frontend-ci.yml)
[![Docker](https://github.com/charles-adedotun/kubepulse/actions/workflows/docker.yml/badge.svg)](https://github.com/charles-adedotun/kubepulse/actions/workflows/docker.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/charles-adedotun/kubepulse)](https://goreportcard.com/report/github.com/charles-adedotun/kubepulse)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/release/charles-adedotun/kubepulse.svg)](https://github.com/charles-adedotun/kubepulse/releases/latest)

Intelligent Kubernetes health monitoring with AI-powered diagnostics, predictive analytics, and auto-remediation.

## Overview

KubePulse is a comprehensive, AI-powered Kubernetes health monitoring platform that combines traditional monitoring with advanced artificial intelligence capabilities. It provides real-time cluster health insights, predictive failure analysis, automated remediation suggestions, and intelligent alert management to eliminate noise and improve reliability.

## Key Features

### 🤖 AI-Powered Intelligence
- **Claude Code Integration**: Direct integration with Claude Code CLI for advanced analysis
- **Diagnostic Analysis**: AI-powered root cause analysis for health check failures
- **Predictive Analytics**: Forecast cluster issues up to 7 days in advance
- **Auto-Remediation**: AI-generated remediation actions with safety validation
- **Smart Alert Management**: Intelligent noise reduction and alert correlation
- **Natural Language Queries**: Chat with your cluster using the AI assistant

### ⚡ Real-Time Monitoring
- **WebSocket Streaming**: Live cluster health updates with automatic cleanup
- **Circuit Breaker Protection**: Resilient AI calls with automatic fallback
- **Error Handling**: Comprehensive error tracking and recovery mechanisms
- **Health Check Engine**: Pod, Node, and Service monitoring with anomaly detection

### 🎯 Advanced Features
- **Plugin Architecture**: Extensible health check system for custom monitoring
- **SRE-Native**: Built-in SLI/SLO tracking with error budget management
- **React Dashboard**: Modern web interface with real-time updates
- **RESTful API**: Comprehensive API for integration and automation
- **Prometheus Metrics**: Native metrics export for observability stacks

## Quick Start

### Prerequisites

- **Go 1.23+** (for building from source)
- **Kubernetes cluster** (v1.28+ recommended)
- **kubectl** configured with cluster access
- **Claude Code CLI** (for AI features) - Optional but recommended
- **Node.js 18+** (for web dashboard development)

### Installation

#### From Source (Recommended)

```bash
# Clone the repository
git clone https://github.com/kubepulse/kubepulse.git
cd kubepulse

# Setup development environment
make setup

# Build the binary (includes frontend)
make build

# Install to your PATH
make install
```

#### Quick Development Setup

```bash
# Start both backend and frontend in development mode
make dev

# Or start services separately:
make run                    # Backend only
make frontend-dev          # Frontend only
```

#### Frontend Development Setup

The KubePulse frontend is built with React, TypeScript, and Vite for a modern development experience.

```bash
# Navigate to frontend directory
cd frontend

# Install dependencies
npm install

# Start development server (with hot reload)
npm run dev

# Build for production
npm run build

# Run type checking
npm run type-check

# Run linting
npm run lint
```

Frontend features:
- **Modern React 18** with TypeScript
- **Real-time WebSocket** integration for live updates
- **Responsive design** with Tailwind CSS
- **Dark/Light theme** support
- **Enhanced UI components** for comprehensive monitoring

### Basic Usage

#### CLI Commands

```bash
# Monitor cluster health (one-time check)
kubepulse monitor

# Continuous monitoring with watch mode
kubepulse monitor --watch

# Monitor specific namespace
kubepulse monitor --namespace production

# Monitor specific context
kubepulse monitor --context production-cluster

# Run specific health check
kubepulse check pod-health

# Start web server with AI features
kubepulse serve --port 8080

# Start server with specific context
kubepulse serve --context staging-cluster

# Diagnose cluster issues with AI
kubepulse diagnose --ai

# Specify custom interval
kubepulse monitor --watch --interval 10s
```

#### Web Dashboard

```bash
# Start the web server
kubepulse serve

# Access the dashboard
open http://localhost:8080
```

The web dashboard provides:
- Real-time cluster health visualization
- AI-powered insights and recommendations
- Interactive health check results
- WebSocket-based live updates
- Multi-cluster context switching

### Multi-Cluster Support

KubePulse supports monitoring multiple Kubernetes clusters from a single interface:

#### Context Switching
- **Web UI**: Use the context selector in the header to switch between clusters
- **CLI**: Use the `--context` flag with any command
- **API**: Use the `/api/v1/contexts/*` endpoints

#### Features
- List all available kubeconfig contexts
- Switch between contexts without restarting
- Maintain separate health history per cluster
- WebSocket updates automatically reflect context changes
- Context information displayed in dashboard

#### Examples
```bash
# List available contexts
kubectl config get-contexts

# Monitor specific context
kubepulse monitor --context production

# Start server and switch contexts via UI
kubepulse serve

# Use different kubeconfig file
kubepulse serve --kubeconfig ~/.kube/other-config
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   CLI (Cobra)   │     │React Dashboard  │     │   REST API      │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │ WebSocket              │
    ┌────┴───────────────────────┴─────────────────────────┴────┐
    │                    Core Monitoring Engine                  │
    │                     (with AI Integration)                  │
    │  ┌─────────────┐ ┌──────────────┐ ┌──────────────────┐  │
    │  │Health Checks│ │Circuit Breaker│ │Error Handler     │  │
    │  └─────────────┘ └──────────────┘ └──────────────────┘  │
    │  ┌─────────────┐ ┌──────────────┐ ┌──────────────────┐  │
    │  │Plugin System│ │Alert Manager │ │SLO Tracker       │  │
    │  └─────────────┘ └──────────────┘ └──────────────────┘  │
    └────────────────────────────┬───────────────────────────────┘
                                 │
    ┌────────────────────────────┴───────────────────────────────┐
    │                      AI Engine                             │
    │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐  │
    │  │Claude Client │ │Predictive    │ │Smart Alerts      │  │
    │  │(with Circuit │ │Analyzer      │ │Manager           │  │
    │  │ Breaker)     │ │              │ │                  │  │
    │  └──────────────┘ └──────────────┘ └──────────────────┘  │
    │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐  │
    │  │Remediation   │ │AI Assistant  │ │Response Parser   │  │
    │  │Engine        │ │              │ │                  │  │
    │  └──────────────┘ └──────────────┘ └──────────────────┘  │
    └────────────────────────────┬───────────────────────────────┘
                                 │
    ┌────────────────────────────┴───────────────────────────────┐
    │                     Data & Integration Layer               │
    │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐  │
    │  │Kubernetes API│ │Metrics Store │ │Claude Code CLI   │  │
    │  └──────────────┘ └──────────────┘ └──────────────────┘  │
    └────────────────────────────────────────────────────────────┘
```

## API Endpoints

### Core Health API

```bash
# Basic health status
GET /api/v1/health

# Cluster health overview
GET /api/v1/health/cluster

# All health check results
GET /api/v1/health/checks

# Specific health check
GET /api/v1/health/checks/{name}

# Prometheus metrics
GET /api/v1/metrics

# Active alerts
GET /api/v1/alerts

# Context management
GET /api/v1/contexts                    # List all contexts
GET /api/v1/contexts/current            # Get current context
POST /api/v1/contexts/switch            # Switch context
{
  "context_name": "production"
}
```

### AI-Powered Endpoints

```bash
# Natural language assistant
POST /api/v1/ai/assistant/query
{
  "query": "Why are my pods failing?"
}

# Predictive insights
GET /api/v1/ai/predictions

# Remediation suggestions
GET /api/v1/ai/remediation/{check}/suggestions

# Execute remediation (with dry-run support)
POST /api/v1/ai/remediation/execute
{
  "action_id": "action-123",
  "dry_run": true
}

# Smart alert insights
GET /api/v1/ai/alerts/insights

# AI cluster insights
GET /api/v1/ai/insights

# AI analysis for specific check
POST /api/v1/ai/analyze/{check}

# AI healing suggestions
POST /api/v1/ai/heal/{check}
```

### WebSocket Endpoint

```bash
# Real-time updates
WS /ws
```

## Built-in Health Checks

### Pod Health Check
- **Monitors**: Pod status, restart counts, container readiness
- **Features**: Configurable restart thresholds, namespace filtering
- **Detects**: CrashLoopBackOff, ImagePullBackOff, OOMKilled, scheduling issues
- **Metrics**: Pod counts, failure rates, restart statistics

### Node Health Check  
- **Monitors**: Node conditions, resource usage, availability
- **Features**: CPU/memory/disk pressure detection, NotReady nodes
- **Detects**: Resource exhaustion, node failures, network issues
- **Metrics**: Node availability, resource utilization

### Service Health Check
- **Monitors**: Service endpoints, port availability, DNS resolution
- **Features**: Endpoint validation, service discovery health
- **Detects**: Service misconfigurations, endpoint failures
- **Metrics**: Service availability, endpoint counts

## Plugin Development

Create custom health checks by implementing the `HealthCheck` interface:

```go
type HealthCheck interface {
    Name() string
    Description() string
    Check(ctx context.Context, client kubernetes.Interface) (CheckResult, error)
    Configure(config map[string]interface{}) error
    Interval() time.Duration
    Criticality() Criticality
}
```

Example custom plugin:

```go
type CustomDNSCheck struct{}

func (c *CustomDNSCheck) Check(ctx context.Context, k8s kubernetes.Interface) (CheckResult, error) {
    // Custom DNS health logic
}
```

## Configuration

KubePulse supports comprehensive configuration through multiple methods:
1. **Configuration file** (`.kubepulse.yaml`)
2. **Environment variables**
3. **Command-line flags**
4. **Runtime configuration** (UI settings from server)

### Configuration File

Copy the example configuration to get started:

```bash
cp .kubepulse.yaml.example ~/.kubepulse.yaml
```

Key configuration sections:

#### Server Configuration
```yaml
server:
  port: 8080              # API server port
  host: ""                # Bind address (empty = all interfaces)
  enable_web: true        # Enable web dashboard
  cors_enabled: true      # Enable CORS
  cors_origins:
    - "*"                 # Allowed origins (* = all)
  read_timeout: 15s       # HTTP read timeout
  write_timeout: 15s      # HTTP write timeout
```

#### UI Configuration
```yaml
ui:
  refresh_interval: 10s        # Dashboard refresh rate
  ai_insights_interval: 30s    # AI insights update interval
  max_reconnect_attempts: 5    # WebSocket reconnection attempts
  reconnect_delay: 3s          # Delay between reconnections
  theme: system                # UI theme: light, dark, system
  features:                    # Feature flags
    ai_insights: true
    predictive_analytics: true
    smart_alerts: true
    node_details: true
```

#### Monitoring Configuration
```yaml
monitoring:
  interval: 30s
  enabled_checks:
    - pod-health
    - node-health
    - service-health
  max_history: 1000
  timeout: 30s
```

#### AI Configuration
```yaml
ai:
  enabled: true
  claude_path: "claude"  # Path to Claude Code CLI
  max_turns: 3
  timeout: 120s
```

### Environment Variables

#### Server Environment Variables
```bash
# Server configuration
KUBEPULSE_PORT=8080                    # Server port
KUBEPULSE_HOST=0.0.0.0                 # Bind address
KUBEPULSE_WEB_ENABLED=true             # Enable web UI
KUBEPULSE_CORS_ENABLED=true            # Enable CORS

# UI configuration
KUBEPULSE_UI_REFRESH=10s               # UI refresh interval
KUBEPULSE_UI_THEME=dark                # UI theme
```

#### Frontend Environment Variables
```bash
# Build-time configuration (Vite)
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/ws
VITE_API_TIMEOUT=30000

# UI intervals (milliseconds)
VITE_REFRESH_INTERVAL=10000
VITE_AI_INSIGHTS_INTERVAL=30000
VITE_MAX_RECONNECT_ATTEMPTS=5
VITE_RECONNECT_DELAY=3000

# Feature flags
VITE_FEATURE_AI_INSIGHTS=true
VITE_FEATURE_PREDICTIVE=true
VITE_FEATURE_SMART_ALERTS=true
VITE_FEATURE_NODE_DETAILS=true
```

#### Kubernetes & Monitoring
```bash
# Kubernetes configuration
KUBECONFIG=/path/to/kubeconfig

# AI configuration  
KUBEPULSE_AI_ENABLED=true
KUBEPULSE_CLAUDE_PATH=/usr/local/bin/claude
KUBEPULSE_AI_TIMEOUT=120s

# Monitoring configuration
KUBEPULSE_INTERVAL=30s
KUBEPULSE_NAMESPACE=production
```

### Command-Line Flags

Command-line flags override configuration file and environment variables:

```bash
# Server command with custom configuration
kubepulse serve \
  --port 9090 \
  --interval 20s \
  --config /custom/path/config.yaml

# Monitor with specific settings
kubepulse monitor \
  --namespace production \
  --interval 10s \
  --checks pod-health,node-health
```

### Configuration Precedence

Configuration is loaded in the following order (later sources override earlier):
1. Default values
2. Configuration file (`.kubepulse.yaml`)
3. Environment variables
4. Command-line flags
5. Runtime configuration (for UI features)

### Frontend Configuration

The frontend can be configured at:
1. **Build time** - Using Vite environment variables
2. **Runtime** - Via server-provided configuration
3. **Window object** - For dynamic configuration

#### Runtime Configuration

The frontend automatically fetches configuration from `/api/v1/config/ui` endpoint.

#### Window Configuration

```html
<script>
  // Configure before app loads
  window.__KUBEPULSE_CONFIG__ = {
    apiBaseUrl: 'https://api.example.com',
    wsUrl: 'wss://api.example.com/ws'
  };
</script>
```

### Claude Code CLI Setup

For full AI functionality, install Claude Code CLI:

```bash
# Install Claude Code CLI (example)
curl -L https://claude.ai/download/cli | sh

# Verify installation
claude --version

# Configure KubePulse to use Claude
export KUBEPULSE_CLAUDE_PATH="claude"
```

## Development

### Building

```bash
# Setup development environment
make setup

# Build for current platform (includes frontend)
make build

# Build for all platforms
make build-all

# Development mode (hot reload)
make dev

# Frontend only
make frontend-dev

# Run tests with coverage
make test

# Run linters and checks
make check

# Clean build artifacts
make clean
```

### Project Structure

```
kubepulse/
├── cmd/kubepulse/          # CLI application
│   └── commands/           # Cobra commands (serve, monitor, diagnose, check)
├── pkg/                    # Public packages
│   ├── ai/                 # AI engine and components
│   │   ├── client.go       # Claude Code CLI integration
│   │   ├── circuit_breaker.go # Resilient AI calls
│   │   ├── smart_alerts.go # Intelligent alert management
│   │   ├── remediation.go  # Auto-remediation engine
│   │   └── predictive.go   # Predictive analytics
│   ├── api/                # REST API and WebSocket server
│   │   ├── server.go       # Main server with WebSocket
│   │   └── ai_handlers.go  # AI-specific endpoints
│   ├── core/               # Core monitoring engine
│   │   ├── engine.go       # Main engine with AI integration
│   │   ├── types.go        # Core data structures
│   │   └── errors.go       # Error handling framework
│   ├── health/             # Built-in health checks
│   │   ├── pod_check.go    # Pod health monitoring
│   │   ├── node_check.go   # Node health monitoring
│   │   └── service_check.go # Service health monitoring
│   ├── alerts/             # Alert management
│   ├── ml/                 # ML anomaly detection
│   ├── slo/                # SLO tracking
│   └── plugins/            # Plugin registry
├── frontend/               # React dashboard
│   ├── src/
│   │   ├── components/     # UI components
│   │   │   ├── dashboard/  # Dashboard specific
│   │   │   ├── layout/     # Layout components
│   │   │   └── ui/         # Reusable UI components
│   │   ├── hooks/          # Custom React hooks
│   │   │   ├── useWebSocket.ts   # WebSocket integration
│   │   │   └── useAIInsights.ts  # AI data fetching
│   │   └── lib/            # Utilities
│   ├── package.json        # Node dependencies
│   └── dist/               # Built frontend assets
├── internal/               # Private packages
├── test/                   # Test files and fixtures
├── Makefile               # Build automation
├── go.mod                 # Go dependencies
└── README.md              # This file
```

## AI Features Deep Dive

### Circuit Breaker Protection

KubePulse includes production-ready circuit breaker protection for all AI operations:

- **Failure Threshold**: Configurable maximum failures before opening circuit
- **Timeout Management**: Prevents hanging AI calls with timeouts
- **State Monitoring**: Real-time circuit breaker state tracking
- **Automatic Recovery**: Smart retry logic with exponential backoff

### Security Features

- **Command Validation**: AI-generated commands are validated before execution
- **Path Allowlisting**: Claude CLI path restricted to known safe locations
- **Prompt Sanitization**: Input sanitization to prevent injection attacks
- **Dry-Run Mode**: Test remediation actions safely before execution

### Error Handling Framework

Comprehensive error handling with:
- **Structured Errors**: Rich error context with categories and severity
- **Recovery Strategies**: Automatic recovery for non-critical failures  
- **Error History**: Persistent error tracking for debugging
- **Health Impact**: Error correlation with cluster health status

## Current Capabilities vs. Roadmap

### ✅ Implemented Features
- [x] **AI-Powered Diagnostics** - Root cause analysis using Claude Code
- [x] **Predictive Analytics** - Failure forecasting based on trends
- [x] **Auto-Remediation** - Safe, AI-generated remediation actions
- [x] **Smart Alert Management** - Noise reduction and correlation
- [x] **React Dashboard** - Modern web interface with real-time updates
- [x] **WebSocket Streaming** - Live cluster health updates
- [x] **Circuit Breaker** - Resilient AI integration
- [x] **Comprehensive APIs** - REST endpoints for all features
- [x] **Natural Language Queries** - Chat with your cluster

### 🚧 Roadmap

- [ ] **Advanced ML Models** - Custom anomaly detection training
- [ ] **Multi-cluster Support** - Federated monitoring across clusters  
- [ ] **Plugin Marketplace** - Community-driven health check plugins
- [ ] **Mobile App** - iOS/Android applications for on-the-go monitoring
- [ ] **Integration Ecosystem** - Slack, Teams, PagerDuty, Datadog integrations
- [ ] **Advanced Analytics** - Cost optimization and capacity planning
- [ ] **Compliance Reporting** - SOC2, PCI-DSS compliance dashboards

## CI/CD

KubePulse uses GitHub Actions for continuous integration and delivery:

- **Go CI**: Runs tests, linting, and builds for Go code
- **Frontend CI**: Runs tests, type checking, and builds for React/TypeScript
- **Security Scanning**: Automated vulnerability scanning with Trivy
- **Release Automation**: Automated releases with GoReleaser

All pull requests must pass CI checks before merging.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

KubePulse is licensed under the Apache License 2.0. See [LICENSE](LICENSE) for details.

## Support

- Documentation: [docs.kubepulse.io](https://docs.kubepulse.io)
- Issues: [GitHub Issues](https://github.com/kubepulse/kubepulse/issues)
- Discussions: [GitHub Discussions](https://github.com/kubepulse/kubepulse/discussions)
- Slack: [Join our community](https://kubepulse.slack.com)