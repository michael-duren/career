#!/usr/bin/env bash
# Runs the visual regression suite (tests/visual) exactly the way CI does, so
# screenshots match the committed baselines on any Linux machine:
#   - a throwaway Postgres container, migrated and seeded from
#     tests/visual/workspace.json (import refuses a non-empty database),
#   - the Go server built from this checkout, serving a fresh `npm run build`,
#   - Playwright inside the official Playwright image for the installed
#     @playwright/test version (same Chromium, fonts and Linux rendering as CI).
#
# Usage:
#   scripts/visual-tests.sh                      # compare against the baselines
#   scripts/visual-tests.sh --update-snapshots   # rewrite the baselines
#   scripts/visual-tests.sh -g timeline          # any other Playwright arguments
#
# Needs Docker (with --network host, so Linux), Go and Node. Set
# VISUAL_SKIP_BUILD=1 to reuse an existing dist/.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$PWD

pw_version=$(node -p "require('@playwright/test/package.json').version")
image=${VISUAL_PLAYWRIGHT_IMAGE:-mcr.microsoft.com/playwright:v$pw_version-noble}
pg_port=${VISUAL_PG_PORT:-55433}
port=${VISUAL_PORT:-18999}
pg_name=career-visual-pg-$$
work=$(mktemp -d)
server_pid=

cleanup() {
  status=$?
  [[ -n $server_pid ]] && kill "$server_pid" 2>/dev/null || true
  docker rm -f "$pg_name" >/dev/null 2>&1 || true
  if [[ $status -ne 0 && -f $work/server.log ]]; then
    mkdir -p test-results-visual
    cp "$work/server.log" test-results-visual/server.log
    echo "Server log: test-results-visual/server.log" >&2
  fi
  rm -rf "$work"
}
trap cleanup EXIT

if curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null 2>&1; then
  echo "port $port is already serving; set VISUAL_PORT to a free port" >&2
  exit 1
fi

docker run -d --rm --name "$pg_name" -p "127.0.0.1:$pg_port:5432" \
  -e POSTGRES_USER=career -e POSTGRES_PASSWORD=career -e POSTGRES_DB=career postgres:18 >/dev/null

export APP_ENV=development
export STATIC_DIR=$root/dist
export DATABASE_URL="postgres://career:career@127.0.0.1:$pg_port/career?sslmode=disable"
export LISTEN_ADDR=127.0.0.1:$port
export PUBLIC_ORIGIN=http://127.0.0.1:$port
export TEST_BASE_URL=$PUBLIC_ORIGIN
export AUTH_USERNAME=visual
# Test-only credential, not a real secret: bcrypt hash of TEST_AUTH_PASSWORD (same pair as the CI e2e job).
export AUTH_PASSWORD_HASH='$2b$10$NQWS/GEO7rCTPyMmHf.PLOpECklQNGzl7oAiMHqF47xVBJdwMM4P2'
export TEST_AUTH_PASSWORD=playwright-ci-password
export JWT_SECRET=disposable-visual-only-secret-32-characters
export OTEL_EXPORTER_OTLP_ENDPOINT=
unset ANTHROPIC_API_KEY WHISPER_URL SCHEDULER_GOOGLE_CLIENT_ID

if [[ ${VISUAL_SKIP_BUILD:-} != 1 ]]; then
  npm run build
fi
go tool templ generate >/dev/null
go build -o "$work/career" ./cmd/api

# Over TCP: during init the image runs a socket-only server that a socket check would accept.
for i in $(seq 1 30); do
  docker exec "$pg_name" pg_isready -h 127.0.0.1 -U career -d career >/dev/null 2>&1 && break
  [[ $i == 30 ]] && { echo "postgres did not start" >&2; exit 1; }
  sleep 1
done
# The seed command keeps a private copy of the archive under ./.migration-private,
# so run it from the scratch directory.
(cd "$work" && ./career migrate >/dev/null && ./career seed --file "$root/tests/visual/workspace.json" --source-store visual-fixture >/dev/null)

(cd "$work" && exec ./career serve >"$work/server.log" 2>&1) &
server_pid=$!
for i in $(seq 1 30); do
  kill -0 "$server_pid" 2>/dev/null || { echo "server exited" >&2; exit 1; }
  curl -fsS "$PUBLIC_ORIGIN/healthz" >/dev/null 2>&1 && break
  [[ $i == 30 ]] && { echo "server did not start" >&2; exit 1; }
  sleep 1
done

docker run --rm --network host --ipc host --user "$(id -u):$(id -g)" \
  -e HOME=/tmp -e CI -e TEST_BASE_URL -e AUTH_USERNAME -e TEST_AUTH_PASSWORD \
  -v "$root:/work" -w /work "$image" \
  npx playwright test -c playwright.visual.config.ts "$@"
