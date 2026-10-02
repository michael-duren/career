#!/usr/bin/env bash
# Run the scheduler browser suite against a fresh, disposable PostgreSQL database.
set -euo pipefail
cd "$(dirname "$0")/.."

pg_root=${SCHEDULER_TEST_PG_ROOT:-postgres://career_dev:career_dev_local@127.0.0.1:5433}
port=${SCHEDULER_TEST_PORT:-4539}
db_name="career_scheduler_e2e_$(node -e 'process.stdout.write(require("node:crypto").randomBytes(8).toString("hex"))')"
admin_url="$pg_root/career_dev?sslmode=disable"
export TEST_DATABASE_URL="$pg_root/$db_name?sslmode=disable"
export DATABASE_URL="$TEST_DATABASE_URL"
export TEST_BASE_URL="http://127.0.0.1:$port"
export PUBLIC_ORIGIN="$TEST_BASE_URL"
export LISTEN_ADDR="127.0.0.1:$port"
export AUTH_USERNAME=admin
export TEST_AUTH_PASSWORD=password123
export AUTH_PASSWORD_HASH='$2b$10$DgkrAm6JRG906Gr0BZZ9C.wEBc9Yg0Wm.QetZAgR.20UsQ12ekaHK'
export JWT_SECRET=disposable-scheduler-e2e-session-secret
export OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4317
export STATIC_DIR=dist

scratch=$(mktemp -d)
server_pid=''
db_created=0
cleanup() {
  if [[ -n "$server_pid" ]]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  if [[ "$db_created" == 1 ]]; then
    psql "$admin_url" -v ON_ERROR_STOP=1 -c "DROP DATABASE $db_name WITH (FORCE)" >/dev/null
  fi
  rm -rf "$scratch"
}
trap cleanup EXIT

# Fail before creating a database if another service already owns this port.
node - "$port" <<'JS'
const net = require('node:net');
const probe = net.createServer();
probe.once('error', error => { console.error(`Scheduler test port unavailable: ${error.message}`); process.exitCode = 1; });
probe.listen(Number(process.argv[2]), '127.0.0.1', () => probe.close());
JS

psql "$admin_url" -v ON_ERROR_STOP=1 -c "CREATE DATABASE $db_name" >/dev/null
db_created=1
go tool templ generate
npm run build
go build -o "$scratch/api" ./cmd/api
"$scratch/api" migrate
"$scratch/api" serve >"$scratch/server.log" 2>&1 &
server_pid=$!
for _ in {1..50}; do
  if ! kill -0 "$server_pid" 2>/dev/null; then cat "$scratch/server.log" >&2; exit 1; fi
  if curl -fsS "$TEST_BASE_URL/login/" >/dev/null 2>&1; then
    sleep 0.2
    if ! kill -0 "$server_pid" 2>/dev/null; then cat "$scratch/server.log" >&2; exit 1; fi
    break
  fi
  sleep 0.2
done
curl -fsS "$TEST_BASE_URL/login/" >/dev/null
npx playwright test tests/e2e/scheduler.spec.ts "$@"
