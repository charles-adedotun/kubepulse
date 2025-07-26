# KubePulse CI Setup

## Overview

KubePulse uses a **simple, focused CI system** with three workflows:

1. **Core CI** (`core-ci.yml`) - Essential checks and builds
2. **Claude Review** (`claude-review.yml`) - AI-powered code review
3. **Release** (`release.yml`) - Automated releases

## Workflows

### Core CI (`core-ci.yml`)

**Triggers**: Push to main, Pull Requests
**Duration**: ~5-8 minutes

**Jobs**:
- **backend-tests** - Go tests, coverage (30% minimum), linting, formatting
- **frontend-tests** - Node.js tests, TypeScript check, linting, build
- **security-scan** - gosec static analysis, govulncheck, npm audit
- **build-validation** - Binary build, Docker image, integration testing

**Key Features**:
- Parallel execution for speed
- Clear pass/fail criteria
- Essential checks only
- Artifact uploads for debugging

### Claude Review (`claude-review.yml`)

**Triggers**: Pull Requests, @claude mentions
**Duration**: ~1-2 minutes

**Features**:
- Real Claude Code action (not mock)
- KubePulse-specific context and instructions
- Structured review format
- Sticky comments (updates on new pushes)
- Security-focused analysis

**Review Areas**:
- Code quality and best practices
- Security vulnerabilities
- Kubernetes patterns and RBAC
- Performance considerations
- Test coverage

### Release (`release.yml`)

**Triggers**: Version tags (v*)
**Duration**: ~3-5 minutes

**Features**:
- GoReleaser integration
- Multi-platform binaries
- Docker image publishing
- Release notes generation

## Branch Protection

**Required Status Checks**:
- `backend-tests`
- `frontend-tests` 
- `security-scan`
- `build-validation`
- `claude-review`

**Settings**:
- Require branches to be up to date
- Require status checks to pass
- Dismiss stale reviews when pushed
- Restrict pushes to matching branches

## Migration from Old System

**Removed Workflows** (10 files):
- `ci.yml` - Replaced by core-ci.yml
- `claude-code-review.yml` - Replaced by simplified claude-review.yml
- `claude.yml` - Merged into claude-review.yml
- `docker.yml` - Merged into core-ci.yml build-validation
- `frontend-ci.yml` - Merged into core-ci.yml frontend-tests
- `go-ci.yml` - Merged into core-ci.yml backend-tests
- `merge-decision.yml` - Removed (1,225 lines of complexity)
- `minimal-ci.yml` - Replaced by core-ci.yml
- `performance.yml` - Basic benchmarks moved to core-ci.yml
- `security-scan.yml` - Merged into core-ci.yml security-scan

**Benefits**:
- **90% reduction** in workflow complexity
- **Faster CI** with parallel execution
- **Clearer responsibilities** - one job per concern
- **Easier maintenance** - simple, readable workflows
- **Real Claude integration** - not mock implementations

## Secrets Required

- `CLAUDE_CODE_OAUTH_TOKEN` - For Claude Code review action
- `GITHUB_TOKEN` - Automatically provided by GitHub
- `CODECOV_TOKEN` - Optional for coverage reporting

## Local Development

**Run tests locally**:
```bash
# Backend
go test -v -race ./...
go vet ./...
golangci-lint run

# Frontend
cd frontend
npm test
npm run type-check
npm run lint
npm run build
```

**Security scan locally**:
```bash
gosec ./...
govulncheck ./...
```

## Troubleshooting

**Common Issues**:

1. **Coverage below 30%** - Add tests for new code
2. **Linting failures** - Run `golangci-lint run --fix`
3. **Formatting issues** - Run `gofmt -s -w .`
4. **Claude review failing** - Check CLAUDE_CODE_OAUTH_TOKEN secret
5. **Docker build fails** - Ensure frontend builds first

**Support**:
- Check workflow logs in GitHub Actions
- Review this documentation
- Ask @claude for help in PR comments