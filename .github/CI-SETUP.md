# KubePulse CI Setup

KubePulse uses a focused CI system with two active workflows:

1. `core-ci.yml`: checks and builds for pushes and pull requests targeting `main`.
2. `release.yml`: GoReleaser publishing for version tags matching `v*`.

Claude Code review automation is disabled.

## Core CI

`core-ci.yml` runs these jobs:

- `backend-tests`: Go tests with race detection and coverage, `golangci-lint`, and `gofmt`.
- `frontend-tests`: frontend install, type check, lint, test placeholder, and production build.
- `security-scan`: `gosec`, SARIF upload, `govulncheck`, and `npm audit`.
- `manifest-validation`: renders base, staging, and production Kubernetes manifests.
- `build-validation`: builds the CLI binary and Docker image.
- `summary`: reports job results.

## Release

`release.yml` runs on version tags and uses GoReleaser.

Required token:

- `GITHUB_TOKEN`, automatically provided by GitHub Actions.

## Local Development

```bash
go test -v -race ./...
go vet ./...

cd frontend
npm ci
npm test
npm run type-check
npm run lint
npm run build
```

## Troubleshooting

- Coverage below 30 percent: add tests for new Go code.
- Linting failures: run `golangci-lint run` locally.
- Formatting issues: run `gofmt -s -w .`.
- Frontend audit failures: update or replace the vulnerable dependency before merging.
