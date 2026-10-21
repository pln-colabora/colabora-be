#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-compose.yaml}"
MIN_AVAILABLE_KIB="${MIN_AVAILABLE_KIB:-262144}"

available_kib="$(awk '/MemAvailable:/ {print $2}' /proc/meminfo)"
if [[ -z "$available_kib" || "$available_kib" -ge "$MIN_AVAILABLE_KIB" ]]; then
  exit 0
fi

logger -t colabora-relay-guard "MemAvailable=${available_kib:-unknown}KiB; stopping Relay/AuthentiK stack to protect COLABORA"
docker compose -f "$COMPOSE_FILE" stop relay authentik-server authentik-worker authentik-postgresql authentik-redis relay-postgres
