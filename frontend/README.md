# KubePulse Frontend

React 19, TypeScript, Vite, and Tailwind dashboard for the KubePulse Go backend.

## Current Surface

- Cluster health overview and health check details.
- Pod, node, service, alert, and metrics views backed by `/api/v1/*` endpoints.
- WebSocket updates from `/ws`.
- AI insight panels that call the backend AI endpoints when the backend can run the configured Claude Code CLI.

## Development

```bash
npm ci
npm run dev
```

The dev server expects the backend at `http://localhost:8080` unless overridden through Vite environment variables.

## Verification

```bash
npm run type-check
npm run lint
npm run build
npm audit --audit-level=moderate
npm test
```

`npm test` is currently a placeholder. There are no frontend unit or end-to-end tests in this repository yet, so CI proof is type-check, lint, production build, and dependency audit.

## Environment

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
VITE_DEV_MODE=true
```

## Deployment

`npm run build` writes the Vite bundle to `frontend/dist`. The main Dockerfile builds this bundle and copies it into the final KubePulse image.

## License

MIT, same as the main repository.
