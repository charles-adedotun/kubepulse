# KubePulse - Claude Development Guide

## Test Commands

### Go Tests
```bash
# Run all tests
make test

# Run Go tests with coverage
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run tests for specific package
go test -v ./pkg/core/...
go test -v ./pkg/ai/...

# Run tests with short flag (skip integration tests)
make test-unit
```

### Frontend Tests
```bash
# Run frontend tests
cd frontend && npm test

# Run tests in watch mode
cd frontend && npm run test:watch

# Run tests with coverage
cd frontend && npm run test:coverage
```

## Lint & Format Commands

### Go
```bash
# Run golangci-lint
make lint

# Format Go code
make fmt

# Run go vet
make vet

# Run all checks (fmt, vet, lint, test)
make check
```

### Frontend
```bash
# Run ESLint
cd frontend && npm run lint

# Run TypeScript type checking
cd frontend && npm run type-check

# Fix lint issues automatically
cd frontend && npm run lint -- --fix
```

## Build Commands

```bash
# Build binary
make build

# Build for all platforms
make build-all

# Build frontend only
make frontend-build

# Clean build artifacts
make clean
```

## Common Issues & Solutions

### SQLite Build Errors
If you get CGO-related errors when building:
```bash
# Ensure CGO is enabled for SQLite
CGO_ENABLED=1 go build ./...
```

### Frontend Dependency Issues
```bash
# Clean install dependencies
cd frontend
rm -rf node_modules package-lock.json
npm install
```

### AI Features Not Working
1. Check if AI is enabled in config
2. Ensure Claude CLI is installed and accessible
3. Check database path permissions
4. Review logs for AI initialization errors

## Development Workflow

1. **Before committing:**
   ```bash
   make check  # Runs fmt, vet, lint, and tests
   ```

2. **Frontend changes:**
   ```bash
   cd frontend
   npm run type-check
   npm run lint
   npm test
   ```

3. **Full verification:**
   ```bash
   make clean
   make build
   make test
   ```

## Key Files to Review
- `pkg/ai/database.go` - AI database schema
- `pkg/core/engine.go` - Core monitoring engine
- `internal/config/config.go` - Configuration structures
- `frontend/src/hooks/useWebSocket.ts` - WebSocket client
- `pkg/api/server.go` - API server and WebSocket handler