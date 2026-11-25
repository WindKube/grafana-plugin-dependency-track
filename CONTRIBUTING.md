# Contributing

## Layout

| Path              | What                                                                |
| ----------------- | ------------------------------------------------------------------- |
| `src/`            | App plugin `windkube-dependencytrack-app` (Scenes pages)            |
| `src/datasource/` | Nested data source `windkube-dependencytrack-datasource` (frontend) |
| `pkg/`            | Data source backend (Go)                                            |
| `provisioning/`   | Provisioning used by the local Grafana                              |
| `tests/`          | Playwright e2e tests                                                |

The data source is nested inside the app. Grafana discovers it from `dist/datasource/plugin.json`, and the backend binary is built to `dist/datasource/gpx_dependencytrack_<os>_<arch>`.

## Prerequisites

- Node.js 24 (see `.nvmrc`)
- pnpm (the exact version is pinned in `package.json` under `packageManager`; `corepack enable` picks it up)
- Go (see `go.mod`) and [Mage](https://magefile.org/)
- Docker with Compose

## Frontend

```bash
pnpm install
pnpm run dev        # build in watch mode
pnpm run build      # production build
pnpm run typecheck
pnpm run lint
pnpm run test:ci    # jest
```

## Backend

```bash
mage -v build:linux   # or build:linuxARM64, build:darwinARM64, ...
mage -v               # all platforms
mage test
golangci-lint run ./...
```

## Running Grafana locally

Build the frontend and the backend for your Docker platform first, then:

```bash
docker compose up --build
```

Grafana runs on http://localhost:3000 with anonymous admin access. Use `GRAFANA_VERSION` to pick another Grafana version.

## E2E tests

```bash
pnpm exec playwright install chromium
docker compose up -d
pnpm run e2e
```

## Pull requests

CI runs typecheck, lint, jest, golangci-lint, Go tests, the backend build, the Grafana API compatibility check and Playwright e2e tests against the supported Grafana versions. Keep it green.

Dependencies are updated by Renovate.
