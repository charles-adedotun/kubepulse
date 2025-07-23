# KubePulse

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

- **Go 1.24.4+** (for building from source)
- **Kubernetes cluster** (v1.28+ recommended)
- **kubectl** configured with cluster access
- **Claude Code CLI** (for AI features) - Optional but recommended
- **Node.js 18+** (for web dashboard development)

### Installation

#### From Source

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

# Run specific health check
kubepulse check pod-health

# Start web server with AI features
kubepulse serve --port 8080

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

### Configuration File

Create a `.kubepulse.yaml` file in your home directory:

```yaml
# Kubernetes configuration
kubeconfig: ~/.kube/config

# Monitoring settings
monitoring:
  interval: 30s
  enabled_checks:
    - pod-health
    - node-health
    - service-health

# AI Configuration
ai:
  enabled: true
  claude_path: "claude"  # Path to Claude Code CLI
  max_turns: 3
  timeout: "120s"
  
# Web server configuration
server:
  port: 8080
  enable_web: true
  cors_enabled: true

# Alert settings
alerts:
  channels:
    - type: slack
      webhook: https://hooks.slack.com/...
    - type: email
      smtp_server: smtp.example.com
      recipients: ["admin@example.com"]

# SLO definitions
slos:
  api-availability:
    sli: availability
    target: 99.9
    window: 30d
    
# Health check specific configuration
health_checks:
  pod-health:
    restart_threshold: 5
    exclude_namespaces: ["kube-system", "kube-public"]
  node-health:
    check_pressure: true
    memory_threshold: 85
    disk_threshold: 90
```

### Environment Variables

```bash
# Kubernetes configuration
KUBECONFIG=/path/to/kubeconfig

# AI configuration  
KUBEPULSE_AI_ENABLED=true
KUBEPULSE_CLAUDE_PATH=/usr/local/bin/claude
KUBEPULSE_AI_TIMEOUT=120s

# Server configuration
KUBEPULSE_PORT=8080
KUBEPULSE_WEB_ENABLED=true

# Monitoring configuration
KUBEPULSE_INTERVAL=30s
KUBEPULSE_NAMESPACE=production
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

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

KubePulse is licensed under the Apache License 2.0. See [LICENSE](LICENSE) for details.

## Support

- Documentation: [docs.kubepulse.io](https://docs.kubepulse.io)
- Issues: [GitHub Issues](https://github.com/kubepulse/kubepulse/issues)
- Discussions: [GitHub Discussions](https://github.com/kubepulse/kubepulse/discussions)
- Slack: [Join our community](https://kubepulse.slack.com)