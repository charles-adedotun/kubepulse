# KubePulse

KubePulse is a Kubernetes health and observability tool with a Go CLI, Go API server, and React dashboard. The root [README](../README.md) is the product entry point: it covers quickstart, commands, checks, architecture, deployment manifests, testing, and current limitations.

## Repository Automation

This `.github/` directory contains the repository automation that supports KubePulse development:

- `workflows/core-ci.yml`: backend tests, frontend checks, security scans, manifest validation, binary build, Docker build, and CI summary.
- `workflows/release.yml`: tagged release publishing through GoReleaser.

See [docs/github-workflows.md](../docs/github-workflows.md) for the detailed workflow reference.
