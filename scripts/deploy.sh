#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "Compose file not found: $COMPOSE_FILE" >&2
  exit 1
fi

if [ ! -f ".env" ]; then
  echo ".env not found. Copy .env.example to .env and update values first." >&2
  exit 1
fi

APP_IMAGE="${APP_IMAGE:-}"
if [ -z "$APP_IMAGE" ]; then
  APP_IMAGE="$(grep '^APP_IMAGE=' .env | cut -d '=' -f2- || true)"
fi

if [ -z "$APP_IMAGE" ]; then
  echo "APP_IMAGE is empty. Set APP_IMAGE in .env or export APP_IMAGE before running." >&2
  exit 1
fi

export APP_IMAGE
export IMAGE_TAG

mkdir -p logs uploads

if [ -n "${GHCR_USERNAME:-}" ] && [ -n "${GHCR_TOKEN:-}" ]; then
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
fi

echo "Deploying image: ${APP_IMAGE}:${IMAGE_TAG}"
docker compose -f "$COMPOSE_FILE" pull app
docker compose -f "$COMPOSE_FILE" up -d --remove-orphans

echo "Current status:"
docker compose -f "$COMPOSE_FILE" ps

echo "Recent app logs:"
docker compose -f "$COMPOSE_FILE" logs --tail=100 app
