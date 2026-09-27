#!/usr/bin/env bash
# Start the full local stack, doing only the steps that are still needed:
# npm deps, .env, PostgreSQL, observability, the Astro build, then the Go server.
# SKIP_OTEL=1 skips the observability containers; without a collector, metrics go to stdout.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -P)

step() { printf '\n==> %s\n' "$*"; }
listening() { (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null; }
env_value() { # Last assignment in .env wins, matching godotenv; quotes stripped.
  sed -n "s/^$1=//p" .env 2>/dev/null | tail -n 1 | sed "s/^['\"]//; s/['\"]\$//"
}

step "npm dependencies"
if [[ ! -f node_modules/.package-lock.json || package-lock.json -nt node_modules/.package-lock.json ]]; then
  npm ci
else
  echo "up to date"
fi

step ".env"
if [[ -e .env || -L .env ]]; then
  echo "found"
else
  ./scripts/setup-env.sh
fi

step "PostgreSQL"
database_url=${DATABASE_URL:-$(env_value DATABASE_URL)}
pg_port=$(sed -n 's#.*@[^:/]*:\([0-9]*\)/.*#\1#p' <<<"$database_url")
pg_port=${pg_port:-5433}
if listening "$pg_port"; then
  echo "already running on port $pg_port"
else
  docker compose up -d --wait postgres
fi

step "Observability (Alloy, Prometheus, Grafana)"
if [[ "${SKIP_OTEL:-}" == 1 ]]; then
  echo "skipped (SKIP_OTEL=1)"
elif listening 4317; then
  echo "collector already running on port 4317"
elif ! docker compose up -d --no-recreate alloy prometheus grafana; then
  echo "warning: observability stack failed to start (see above); continuing without it" >&2
fi
if [[ "${SKIP_OTEL:-}" == 1 ]] || ! listening 4317; then
  # No collector: print metrics to stdout rarely instead of failing every export.
  export OTEL_EXPORTER_OTLP_ENDPOINT= OTEL_METRIC_EXPORT_INTERVAL=600000
  echo "no collector on port 4317; metrics go to stdout every 10 minutes"
fi

step "Frontend build"
# Rebuild when dist is missing or any frontend input is newer than the last build.
marker=dist/.build-stamp
if [[ ! -f "$marker" ]] || [[ -n "$(find src public astro.config.mjs tailwind.config.mjs tsconfig.json package.json package-lock.json -newer "$marker" -print -quit 2>/dev/null)" ]]; then
  npm run build
  touch "$marker"
else
  echo "dist is current"
fi

step "Go server"
listen=${LISTEN_ADDR:-$(env_value LISTEN_ADDR)}
listen=${listen:-127.0.0.1:8080}
port=${listen##*:}
if listening "$port"; then
  # Replace a previous dev server from this checkout; refuse to touch anything else.
  pids=$(ss -ltnpH "sport = :$port" 2>/dev/null | grep -o 'pid=[0-9]*' | cut -d= -f2 | sort -u)
  ours=true
  for pid in $pids; do
    [[ "$(readlink "/proc/$pid/cwd" 2>/dev/null)" == "$root" ]] || ours=false
  done
  if [[ -z "$pids" || "$ours" != true ]]; then
    echo "Port $port is in use by another program. Stop it or set LISTEN_ADDR." >&2
    exit 1
  fi
  echo "stopping the previous dev server on port $port"
  kill $pids
  for _ in $(seq 50); do listening "$port" || break; sleep 0.1; done
fi

go tool templ generate
go run ./cmd/api migrate
public_origin=${PUBLIC_ORIGIN:-$(env_value PUBLIC_ORIGIN)}
echo "Serving on ${public_origin:-http://$listen}"
exec go run ./cmd/api serve
