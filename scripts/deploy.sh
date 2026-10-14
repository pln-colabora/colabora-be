#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/colabora-be}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
HEALTHCHECK_URL="${HEALTHCHECK_URL:-http://127.0.0.1:${GOLANG_PORT:-8888}/health}"

cd "$DEPLOY_DIR"
export IMAGE_TAG="${IMAGE_TAG:?IMAGE_TAG must be set}"

echo "==> Pulling ${IMAGE_TAG}"
docker compose -f "$COMPOSE_FILE" pull app

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
