#!/usr/bin/env bash
set -euo pipefail

ENVIRONMENT="${1:-}"
if [[ "$ENVIRONMENT" != "prod" && "$ENVIRONMENT" != "dev" ]]; then
  echo "Usage: $0 [prod|dev]" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

PRODUCT_NAME="$(awk 'NR==1 {print $0}' .project)"
VERSION="$(awk 'NR==2 {print $0}' .project)"

BACKEND_IMAGE="${PRODUCT_NAME}-backend"
FRONTEND_IMAGE="${PRODUCT_NAME}-frontend"

BACKEND_TAG="${BACKEND_IMAGE}:${VERSION}"
FRONTEND_TAG="${FRONTEND_IMAGE}:${VERSION}"

PACKAGE_DIR="dist/${PRODUCT_NAME}-${VERSION}-${ENVIRONMENT}"
IMAGES_DIR="${PACKAGE_DIR}/images"
BACKEND_CONFIG_DIR="${PACKAGE_DIR}/backend/config"
MIGRATIONS_OUT_DIR="${PACKAGE_DIR}/migrations"

rm -rf "$PACKAGE_DIR"
mkdir -p "$IMAGES_DIR" "$BACKEND_CONFIG_DIR" "$MIGRATIONS_OUT_DIR"

# Compose
cp "docker/docker-compose.yml" "${PACKAGE_DIR}/docker-compose.yml"

# Compose env (docker compose will auto-load .env)
cat > "${PACKAGE_DIR}/.env" <<EOF
BACKEND_IMAGE=${BACKEND_IMAGE}
BACKEND_VERSION=${VERSION}
FRONTEND_IMAGE=${FRONTEND_IMAGE}
FRONTEND_VERSION=${VERSION}
EOF

# Config
cp "docker/backend/config/casbin_model.conf" "${BACKEND_CONFIG_DIR}/casbin_model.conf"
cp "docker/backend/config/config.yaml" "${BACKEND_CONFIG_DIR}/config.yaml"

# Migrations (for offline upgrade + auto-migrate)
cp migrations/*.sql migrations/atlas.sum "${MIGRATIONS_OUT_DIR}/"

# Manage script
cp "hack/script/manage.sh" "${PACKAGE_DIR}/manage.sh"
chmod +x "${PACKAGE_DIR}/manage.sh"

# Images
if ! docker image inspect "$BACKEND_TAG" >/dev/null 2>&1; then
  echo "Backend image not found: $BACKEND_TAG" >&2
  echo "Hint: run 'make buildall' first." >&2
  exit 1
fi
if ! docker image inspect "$FRONTEND_TAG" >/dev/null 2>&1; then
  echo "Frontend image not found: $FRONTEND_TAG" >&2
  echo "Hint: run 'make buildall' first." >&2
  exit 1
fi

docker save "$BACKEND_TAG" > "${IMAGES_DIR}/backend.tar"
docker save "$FRONTEND_TAG" > "${IMAGES_DIR}/frontend.tar"

# Archive
tar -C dist -czf "dist/${PRODUCT_NAME}-${VERSION}-${ENVIRONMENT}.tar.gz" "${PRODUCT_NAME}-${VERSION}-${ENVIRONMENT}"
echo "Package created: dist/${PRODUCT_NAME}-${VERSION}-${ENVIRONMENT}.tar.gz"
