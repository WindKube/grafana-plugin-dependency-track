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

This starts:

| Service        | URL                   | Notes                                      |
| -------------- | --------------------- | ------------------------------------------ |
| `grafana`      | http://localhost:3000 | Anonymous admin, plugin from `dist/`       |
| `dt-apiserver` | http://localhost:8081 | Dependency-Track 5.x API server            |
| `dt-frontend`  | http://localhost:8082 | Dependency-Track UI, `admin`/`grafana-dev` |
| `dt-postgres`  | –                     | Dependency-Track database                  |
| `dt-seed`      | –                     | One-shot seed, see below                   |

`dev/seed/seed.sh` runs once against a fresh Dependency-Track:

- sets the admin password to `grafana-dev`;
- creates a `Grafana` team with `VIEW_PORTFOLIO`, `VIEW_VULNERABILITY` and `VIEW_POLICY_VIOLATION`, plus an API key;
- creates three policies (FAIL, WARN and INFO);
- uploads the CycloneDX BOMs from `dev/seed/boms` (tags come from the `seed:tag` metadata properties);
- attaches internal vulnerabilities of every severity to some components, then evaluates policies and refreshes metrics.

The API key is written to the `dt-seed` volume, and `provisioning/datasources/datasources.yml` reads it with `$__file{}`.

Overrides: use `GRAFANA_VERSION` to pick another Grafana version and `DT_VERSION` for another Dependency-Track version. To start over with a fresh Dependency-Track, run `docker compose down -v`.

## E2E tests

```bash
pnpm exec playwright install chromium
docker compose up -d
pnpm run e2e
```

## Pull requests

CI runs typecheck, lint, jest, golangci-lint, Go tests, the backend build, the Grafana API compatibility check and Playwright e2e tests against the supported Grafana versions. Keep it green.

Dependencies are updated by Renovate.
