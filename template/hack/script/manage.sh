#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

IMAGES_DIR="${SCRIPT_DIR}/images"
MIGRATIONS_DIR="${SCRIPT_DIR}/migrations"
DOCKER_COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.yml"

# Auto load .env if present (docker compose will load it too, but we also need it for migrate/smoke).
if [[ -f "${SCRIPT_DIR}/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "${SCRIPT_DIR}/.env"
  set +a
fi

die() {
  echo "ERROR: $*" >&2
  exit 1
}

require_file() {
  [[ -f "$1" ]] || die "Missing file: $1"
}

load_images() {
  require_file "${IMAGES_DIR}/backend.tar"
  require_file "${IMAGES_DIR}/frontend.tar"
  echo "Loading Docker images..."
  docker load < "${IMAGES_DIR}/backend.tar"
  docker load < "${IMAGES_DIR}/frontend.tar"
  echo "Docker images loaded."
}

migrate_db() {
  [[ -n "${DATABASE_URL:-}" ]] || die "DATABASE_URL is required for migration (example: postgres://user:pass@127.0.0.1:5432/projecttemplate?sslmode=disable)"
  [[ -d "${MIGRATIONS_DIR}" ]] || die "Missing migrations dir: ${MIGRATIONS_DIR}"
  require_file "${MIGRATIONS_DIR}/atlas.sum"

  echo "Applying migrations via Atlas..."
  docker run --rm --network host \
    -v "${MIGRATIONS_DIR}:/migrations" \
    arigaio/atlas:latest \
    migrate apply --dir file:///migrations --url "${DATABASE_URL}"
  echo "Migrations applied."
}

start_services() {
  require_file "${DOCKER_COMPOSE_FILE}"
  echo "Starting services..."
  docker compose -f "${DOCKER_COMPOSE_FILE}" up --no-build --pull never -d
  echo "Services started."
}

stop_services() {
  require_file "${DOCKER_COMPOSE_FILE}"
  echo "Stopping services..."
  docker compose -f "${DOCKER_COMPOSE_FILE}" down
  echo "Services stopped."
}

restart_services() {
  stop_services
  start_services
}

smoke_readyz() {
  local backend_port="${BACKEND_PORT:-8080}"
  local url="${READY_URL:-http://127.0.0.1:${backend_port}/readyz}"

  echo "Waiting for readyz: ${url}"
  for i in $(seq 1 60); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      echo "readyz ok"
      return 0
    fi
    sleep 1
  done

  echo "readyz failed, dumping status/logs"
  docker compose -f "${DOCKER_COMPOSE_FILE}" ps || true
  docker compose -f "${DOCKER_COMPOSE_FILE}" logs --tail=200 || true
  return 1
}

upgrade() {
  # Offline upgrade flow: load images -> migrate -> up -> smoke
  load_images
  migrate_db
  start_services
  smoke_readyz
}

usage() {
  cat <<'EOF'
Usage: ./manage.sh <command>

Commands:
  load      Load docker images from ./images/*.tar
  migrate   Apply DB migrations using Atlas (requires DATABASE_URL)
  start     docker compose up -d
  stop      docker compose down
  restart   stop then start
  smoke     Poll /readyz until ready or timeout
  upgrade   load -> migrate -> start -> smoke

Notes:
  - This script will source ./.env if present.
  - DATABASE_URL is only required for migrate/upgrade.
EOF
}

main() {
  case "${1:-}" in
    load) load_images ;;
    migrate) migrate_db ;;
    start) start_services ;;
    stop) stop_services ;;
    restart) restart_services ;;
    smoke) smoke_readyz ;;
    upgrade) upgrade ;;
    ""|-h|--help|help) usage ;;
    *) usage; die "Unknown command: $1" ;;
  esac
}

main "$@"
