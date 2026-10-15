#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/colabora-be}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
HEALTHCHECK_URL="${HEALTHCHECK_URL:-http://127.0.0.1:${GOLANG_PORT:-8888}/health}"

cd "$DEPLOY_DIR"
export IMAGE_TAG="${IMAGE_TAG:?IMAGE_TAG must be set}"

echo "==> Pulling ${IMAGE_TAG}"
docker compose -f "$COMPOSE_FILE" pull app

echo "==> Starting deployment dependencies"
docker compose -f "$COMPOSE_FILE" up -d postgres garage

echo "==> Waiting for PostgreSQL"
database_ready=false
for i in $(seq 1 30); do
  if docker compose -f "$COMPOSE_FILE" exec -T postgres \
    sh -c 'pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"' > /dev/null 2>&1; then
    database_ready=true
    break
  fi
  sleep 2
done

if [[ "$database_ready" != "true" ]]; then
  echo "PostgreSQL did not become ready in time."
  docker compose -f "$COMPOSE_FILE" logs --tail=50 postgres
  exit 1
fi

echo "==> Running migrations against the new image (before cutover)"
docker compose -f "$COMPOSE_FILE" run --rm app ./server --migrate:run

echo "==> Rolling out new app container"
docker compose -f "$COMPOSE_FILE" up -d

echo "==> Restarting nginx (picks up config changes; re-resolves app's IP)"
docker compose -f "$COMPOSE_FILE" restart nginx

echo "==> Waiting for health check"
for i in $(seq 1 10); do
  if curl -fsS "$HEALTHCHECK_URL" > /dev/null; then
    echo "Deploy OK (${IMAGE_TAG})"
    exit 0
  fi
  sleep 3
done

echo "Health check FAILED for ${IMAGE_TAG}."
echo "Manual rollback: IMAGE_TAG=<previous-sha-tag> ./scripts/deploy.sh"
exit 1
