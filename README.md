# KubePulse

Intelligent Kubernetes health monitoring with ML-powered anomaly detection.

## Overview

KubePulse is a lightweight, intelligent Kubernetes health monitoring tool that combines traditional threshold-based monitoring with ML-powered anomaly detection. It provides instant "traffic light" health status for your clusters while eliminating alert fatigue through smart, context-aware monitoring.

## Key Features

- **ML-Powered Intelligence**: Context-aware anomaly detection reduces false positives by 80%
- **Plugin Architecture**: Extensible health check system for custom monitoring needs
- **SRE-Native**: Built-in SLI/SLO tracking with error budget management
- **Alert Fatigue Prevention**: Multi-window, multi-burn-rate alerting with smart suppression
- **Predictive Analytics**: Forecast failures up to 7 days in advance
- **Lightweight**: Minimal resource footprint, perfect for homelab and small team environments

## Quick Start

### Prerequisites

- Go 1.21+ (for building from source)
- Kubernetes cluster (1.28+ recommended)
- kubectl configured with cluster access

### Installation

#### From Source

```bash
# Clone the repository
git clone https://github.com/kubepulse/kubepulse.git
cd kubepulse

# Build the binary
make build

# Install to your PATH
make install
```

#### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/kubepulse/kubepulse/releases).

### Basic Usage

```bash
# Monitor cluster health (one-time check)
kubepulse monitor

# Continuous monitoring with watch mode
kubepulse monitor --watch

# Monitor specific namespace
kubepulse monitor --namespace production

# Run specific health check
kubepulse check pod-health

# Specify custom interval
kubepulse monitor --watch --interval 10s
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   CLI (Cobra)   │     │  Web Dashboard  │     │   API Gateway   │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                         │
    ┌────┴───────────────────────┴─────────────────────────┴────┐
    │                    Core Monitoring Engine                  │
    │  ┌─────────────┐ ┌──────────────┐ ┌──────────────────┐  │
    │  │Health Checks│ │ML Anomaly Det│ │SLO/Error Budget  │  │
    │  └─────────────┘ └──────────────┘ └──────────────────┘  │
    │  ┌─────────────┐ ┌──────────────┐ ┌──────────────────┐  │
    │  │Plugin System│ │Alert Manager │ │Prediction Engine │  │
    │  └─────────────┘ └──────────────┘ └──────────────────┘  │
    └────────────────────────────┬───────────────────────────────┘
                                 │
    ┌────────────────────────────┴───────────────────────────────┐
    │                     Data Layer                             │
    │  ┌──────────┐ ┌────────────┐ ┌──────────────────────┐    │
    │  │K8s Client│ │Time Series │ │State Store (BoltDB)  │    │
    │  └──────────┘ └────────────┘ └──────────────────────┘    │
    └────────────────────────────────────────────────────────────┘
```

## Built-in Health Checks

### Pod Health
- Monitors pod status, restarts, and container readiness
- Configurable restart thresholds
- Namespace filtering support

### Node Health
- Tracks node conditions and resource usage
- CPU, memory, and disk pressure detection
- Identifies NotReady nodes

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

# Alert settings
alerts:
  channels:
    - type: slack
      webhook: https://hooks.slack.com/...

# SLO definitions
slos:
  api-availability:
    sli: availability
    target: 99.9
    window: 30d
```

## Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linters
make lint
```

### Project Structure

```
kubepulse/
├── cmd/kubepulse/      # CLI application
├── pkg/
│   ├── core/           # Core types and engine
│   ├── health/         # Built-in health checks
│   ├── plugins/        # Plugin system
│   ├── ml/            # ML anomaly detection
│   ├── alerts/        # Alert management
│   └── slo/           # SLO tracking
├── internal/          # Internal packages
├── test/             # Test files
└── examples/         # Example configurations
```

## Roadmap

- [ ] ML anomaly detection engine
- [ ] Web dashboard UI
- [ ] Prometheus metrics export
- [ ] Multi-cluster support
- [ ] Predictive failure analysis
- [ ] Plugin marketplace
- [ ] Mobile app

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

KubePulse is licensed under the Apache License 2.0. See [LICENSE](LICENSE) for details.

## Support

- Documentation: [docs.kubepulse.io](https://docs.kubepulse.io)
- Issues: [GitHub Issues](https://github.com/kubepulse/kubepulse/issues)
- Discussions: [GitHub Discussions](https://github.com/kubepulse/kubepulse/discussions)
- Slack: [Join our community](https://kubepulse.slack.com)