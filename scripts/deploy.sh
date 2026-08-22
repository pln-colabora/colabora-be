#!/usr/bin/env bash
set -euo pipefail

cd /opt/colabora-be
export IMAGE_TAG="${IMAGE_TAG:?IMAGE_TAG must be set}"

echo "==> Pulling ${IMAGE_TAG}"
docker compose -f docker-compose.prod.yml pull app

echo "==> Running migrations against the new image (before cutover)"
docker compose -f docker-compose.prod.yml run --rm app ./server --migrate:run

echo "==> Rolling out new app container"
docker compose -f docker-compose.prod.yml up -d

echo "==> Waiting for health check"
for i in $(seq 1 10); do
  if curl -fsS "http://127.0.0.1:${GOLANG_PORT:-8888}/health" > /dev/null; then
    echo "Deploy OK (${IMAGE_TAG})"
    exit 0
  fi
  sleep 3
done

echo "Health check FAILED for ${IMAGE_TAG}."
echo "Manual rollback: IMAGE_TAG=<previous-sha-tag> ./scripts/deploy.sh"
exit 1
