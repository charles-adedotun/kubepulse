# GitHub Workflows

This document summarizes the active GitHub Actions setup for KubePulse.

## Workflow Files

| Workflow | File | Trigger | Purpose |
| --- | --- | --- | --- |
| Core CI | `.github/workflows/core-ci.yml` | Pushes and pull requests targeting `main` | Backend tests, frontend checks, security scans, manifest validation, binary build, Docker build. |
| Release | `.github/workflows/release.yml` | Tags matching `v*` | Runs GoReleaser for tagged releases. |

## Core CI

`core-ci.yml` is the main quality gate. It uses Go 1.25.10 and Node.js 20.

Jobs:

- `backend-tests`: downloads Go modules, runs `go test -v -race -coverprofile=coverage.out ./...`, enforces minimum total coverage of 30 percent, runs `golangci-lint`, and checks `gofmt -s`.
- `frontend-tests`: runs `npm ci`, `npm run type-check`, `npm run lint`, the placeholder `npm test`, and `npm run build` in `frontend/`.
- `security-scan`: runs `gosec`, uploads SARIF results, runs `govulncheck`, and runs `npm audit --audit-level=moderate`.
- `manifest-validation`: renders base, staging, and production Kubernetes manifests with `kubectl kustomize`.
- `build-validation`: builds the Linux binary, checks `--version` and `--help`, builds the Docker image, and checks the container `--version`.
- `summary`: writes a GitHub Step Summary showing job results.

## Release

`release.yml` runs on version tags matching `v*`.

It:

- Checks out the full repository history.
- Sets up Go and Node.js.
- Installs frontend dependencies.
- Runs GoReleaser with `release --clean`.

Required token:

- `GITHUB_TOKEN`, provided by GitHub Actions.

## Local Equivalents

Run these commands before opening or updating a pull request:

```bash
go test -v -race ./...
go vet ./...

cd frontend
npm ci
npm run type-check
npm run lint
npm test
npm run build
npm audit --audit-level=moderate
```

For security parity with CI, install and run:

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

gosec ./...
govulncheck ./...
```

## Notes

- Claude Code review automation is intentionally disabled in this repository.
- Older workflow names such as `minimal-ci.yml`, `claude.yml`, `claude-review.yml`, or `merge-decision.yml` are no longer active.
- The frontend test script is intentionally a placeholder today; CI relies on type checking, linting, production build, and audit until component or end-to-end tests are added.
