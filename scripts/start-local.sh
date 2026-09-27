#!/usr/bin/env bash
# Start the full local stack, doing only the steps that are still needed:
# npm deps, .env, PostgreSQL, observability, the Astro build, then the Go server.
# SKIP_OTEL=1 skips the observability containers; without a collector, metrics go to stdout.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd -P)
export COMPOSE_FILE=${COMPOSE_FILE:-compose/docker-compose.yml}

step() { printf '\n==> %s\n' "$*"; }
die() { echo "error: $*" >&2; exit 1; }

# Reports whether anything listens on the TCP port, on any local address.
listening() {
  if command -v ss >/dev/null; then
    [[ -n "$(ss -ltnH "sport = :$1" 2>/dev/null)" ]]
  else
    (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null
  fi
}

# Reads KEY from .env the way godotenv does: last assignment wins, optional
# `export`, quoted values kept whole, inline comments dropped from bare values.
env_value() {
  [[ -f .env ]] || return 0
  awk -v key="$1" '
    match($0, "^[[:space:]]*(export[[:space:]]+)?" key "[[:space:]]*=[[:space:]]*") {
      v = substr($0, RLENGTH + 1)
      q = substr(v, 1, 1)
      if (q == "\"" || q == "\047") {
        rest = substr(v, 2); end = index(rest, q)
        v = end ? substr(rest, 1, end - 1) : rest
      } else {
        sub(/[[:space:]]+#.*$/, "", v); sub(/[[:space:]]+$/, "", v)
      }
      found = v
    }
    END { print found }' .env
}
# Environment variables win over .env, even when set to empty, as with godotenv.
env_has() { [[ -f .env ]] && grep -Eq "^[[:space:]]*(export[[:space:]]+)?$1[[:space:]]*=" .env; }
defined() { [[ -n ${!1+x} ]] || env_has "$1"; }
setting() { if [[ -n ${!1+x} ]]; then printf '%s\n' "${!1}"; else env_value "$1"; fi; }
local_host() { [[ "$1" == localhost || "$1" == 127.0.0.1 || "$1" == ::1 || "$1" == "[::1]" ]]; }

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

# Splits host[:port] (IPv6 in brackets) into $host and $port.
split_host() {
  if [[ "$1" == \[* ]]; then host=${1%%]*}]; else host=${1%%:*}; fi
  port=${1#"$host"}; port=${port#:}
}

step "PostgreSQL"
# Go's development default when DATABASE_URL is empty.
database_url=$(setting DATABASE_URL)
database_url=${database_url:-postgres://career_dev:career_dev_local@127.0.0.1:5433/career_dev}
authority=${database_url#*://}; authority=${authority#*@}; authority=${authority%%/*}
split_host "$authority"
pg_host=${host:-127.0.0.1}; pg_port=${port:-5432}
if ! local_host "$pg_host"; then
  echo "using remote database at $pg_host"
elif listening "$pg_port"; then
  echo "already running on port $pg_port"
else
  POSTGRES_LOCAL_PORT=$pg_port docker compose up -d --wait postgres
fi

step "Observability (Alloy, Prometheus, Grafana)"
# Same precedence as internal/config: the metrics-specific endpoint, then the
# general one, then the local collector. A defined but empty value means stdout.
if defined OTEL_EXPORTER_OTLP_METRICS_ENDPOINT; then
  endpoint=$(setting OTEL_EXPORTER_OTLP_METRICS_ENDPOINT)
elif defined OTEL_EXPORTER_OTLP_ENDPOINT; then
  endpoint=$(setting OTEL_EXPORTER_OTLP_ENDPOINT)
else
  endpoint=http://localhost:4317
fi
authority=${endpoint#*://}; authority=${authority%%/*}
split_host "$authority"
otel_host=$host; otel_port=${port:-4317}
if [[ -z "$endpoint" ]]; then
  echo "metrics go to stdout (endpoint set empty)"
elif ! local_host "$otel_host"; then
  echo "using configured collector at $endpoint"
else
  if [[ "${SKIP_OTEL:-}" == 1 ]]; then
    echo "skipped (SKIP_OTEL=1)"
  elif listening "$otel_port"; then
    echo "collector already running on port $otel_port"
  elif [[ "$otel_port" != 4317 ]]; then
    echo "warning: nothing listens on $endpoint and the local stack only serves port 4317" >&2
  elif ! docker compose up -d --no-recreate alloy prometheus grafana; then
    echo "warning: observability stack failed to start (see above); continuing without it" >&2
  fi
  if ! listening "$otel_port"; then
    # No collector: print metrics to stdout rarely instead of failing every export.
    export OTEL_EXPORTER_OTLP_METRICS_ENDPOINT= OTEL_EXPORTER_OTLP_ENDPOINT= OTEL_METRIC_EXPORT_INTERVAL=600000
    echo "no collector on port $otel_port; metrics go to stdout every 10 minutes"
  fi
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
listen=$(setting LISTEN_ADDR)
listen=${listen:-127.0.0.1:8080}
port=${listen##*:}
if listening "$port"; then
  # Replace a previous dev server from this checkout; refuse to touch anything else.
  command -v ss >/dev/null || die "port $port is in use and ss is not installed to identify the owner"
  pids=$(ss -ltnpH "sport = :$port" 2>/dev/null | grep -o 'pid=[0-9]*' | cut -d= -f2 | sort -u || true)
  [[ -n "$pids" ]] || die "port $port is in use by a process this user cannot inspect. Stop it or set LISTEN_ADDR."
  for pid in $pids; do
    [[ "$(readlink "/proc/$pid/cwd" 2>/dev/null)" == "$root" ]] ||
      die "port $port is in use by another program (pid $pid). Stop it or set LISTEN_ADDR."
  done
  echo "stopping the previous dev server on port $port"
  kill $pids 2>/dev/null || true
  for _ in $(seq 50); do listening "$port" || break; sleep 0.1; done
  ! listening "$port" || die "the previous dev server on port $port did not stop"
fi

go tool templ generate
go run ./cmd/api migrate
public_origin=$(setting PUBLIC_ORIGIN)
echo "Serving on ${public_origin:-http://$listen}"
exec go run ./cmd/api serve
