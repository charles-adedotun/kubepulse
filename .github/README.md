# KubePulse

KubePulse is a Kubernetes health and observability tool with a Go CLI, Go API server, and React dashboard. The root [README](../README.md) is the product entry point: it covers quickstart, commands, checks, architecture, deployment manifests, testing, and current limitations.

## Repository Automation

This `.github/` directory contains the repository automation that supports KubePulse development:

- `workflows/core-ci.yml`: backend tests, frontend checks, security scans, manifest validation, binary build, Docker build, and CI summary.
- `workflows/claude-review.yml`: optional Claude Code pull request review and `@claude` issue-comment assistance.
- `workflows/release.yml`: tagged release publishing through GoReleaser.

See [docs/github-workflows.md](../docs/github-workflows.md) for the detailed workflow reference.

## Required Secrets

- `CLAUDE_CODE_OAUTH_TOKEN`: required only for the optional Claude review workflow.
- `GITHUB_TOKEN`: provided by GitHub Actions for release and repository operations.

If the Claude review token is absent or invalid, the product code and ordinary CI can still be evaluated through `core-ci.yml`.
