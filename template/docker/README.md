# Docker (Unified)

This repo uses a single Docker layout under `docker/` for local dev, staging and production.
Environment differences are handled via `.env` / environment variables.

## Files

- `docker/docker-compose.yml`: unified compose file
- `docker/.env.example`: example env file (copy to `docker/.env` on the target machine)
- `docker/backend/`: backend Dockerfile + default config
- `docker/frontend/`: frontend Dockerfile + nginx config

## Local Infra (Postgres/Redis/NATS)

Start infra containers only (no app containers):

```bash
make infra-up
make infra-ps
```

Stop:

```bash
make infra-down
```

Notes:
- Infra is started with compose profile `local` (see `docker/docker-compose.yml`).
- Infra ports are bound to `127.0.0.1` by default.

## Run DoorX With docker compose

### 1) Prepare env

Create `docker/.env` based on `docker/.env.example`.

At minimum you must set:
- `BACKEND_IMAGE` / `BACKEND_VERSION`
- `FRONTEND_IMAGE` / `FRONTEND_VERSION`
- config.yaml 对应字段 (for the backend container)
- config.yaml 对应字段 (recommended)

### 2) Start

From repo root:

```bash
docker compose --project-directory docker -f docker/docker-compose.yml up -d
```

Health checks:

```bash
curl -fsS http://127.0.0.1:${BACKEND_PORT:-8080}/healthz
curl -fsS http://127.0.0.1:${BACKEND_PORT:-8080}/readyz
```

OpenAPI (dev-only):
- Path is always `/openapi.yaml` on the backend HTTP port (example: `http://127.0.0.1:${BACKEND_PORT:-8080}/openapi.yaml`).
- It is disabled by default in `docker/backend/config/config.yaml` (`dev.enable_openapi: false`).
- Enable via env in `docker/.env` for compose:
  - `config.yaml 对应字段=true`
  - `config.yaml 对应字段=./openapi.yaml` (path inside backend container working dir)

### 3) Stop

```bash
docker compose --project-directory docker -f docker/docker-compose.yml down
```

## Common Environment Switches

- Use external Redis:
  - `config.yaml 对应字段=external`
  - `config.yaml 对应字段=<host:port>`
- Allow single-node fallback when Redis is unstable:
  - `config.yaml 对应字段=auto`
- Disable NATS:
  - `config.yaml 对应字段=disabled`
- Embedded NATS (single process):
  - `config.yaml 对应字段=embedded`
  - `config.yaml 对应字段=true`
  - `config.yaml 对应字段=/some/persistent/path` (optional)
