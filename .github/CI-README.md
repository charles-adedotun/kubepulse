# KubePulse CI/CD System

KubePulse uses GitHub Actions for build, test, security, manifest validation, and release automation.

## Active Workflows

- `core-ci.yml`: pull request and `main` branch validation.
- `release.yml`: release automation for `v*` tags.

Claude Code review automation is disabled.

## Core CI Coverage

The main CI workflow validates:

- Go backend tests, race checks, coverage, linting, and formatting.
- Frontend dependency install, type checking, linting, test placeholder, build, and audit.
- Static and vulnerability scanning with `gosec`, SARIF upload, `govulncheck`, and `npm audit`.
- Kubernetes manifest rendering for base, staging, and production overlays.
- Linux binary and Docker image builds.

## Local Checks

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

## Release

Tagged releases use GoReleaser through `.github/workflows/release.yml`.
