#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

scratch=$(mktemp -d)
listener_pid=''
cleanup() {
  if [[ -n "$listener_pid" ]]; then kill "$listener_pid" 2>/dev/null || true; fi
  rm -rf "$scratch"
}
trap cleanup EXIT

cat >"$scratch/psql" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"$TEST_PSQL_LOG"
exit 1
STUB
chmod +x "$scratch/psql"
export TEST_PSQL_LOG="$scratch/psql.log"
export PATH="$scratch:$PATH"

# A failed CREATE must not authorize cleanup to drop an existing database.
if SCHEDULER_TEST_PORT=0 bash scripts/verify-scheduler-e2e.sh >"$scratch/create.log" 2>&1; then
  echo 'expected failed database creation' >&2; exit 1
fi
if grep -q 'DROP DATABASE' "$TEST_PSQL_LOG"; then
  echo 'runner dropped a database it did not create' >&2; exit 1
fi

# An occupied port must be rejected before the runner touches PostgreSQL.
node -e 'const s=require("node:net").createServer(); s.listen(0,"127.0.0.1",()=>console.log(s.address().port))' >"$scratch/port" &
listener_pid=$!
for _ in {1..50}; do [[ -s "$scratch/port" ]] && break; sleep 0.1; done
port=$(cat "$scratch/port")
: >"$TEST_PSQL_LOG"
if SCHEDULER_TEST_PORT="$port" bash scripts/verify-scheduler-e2e.sh >"$scratch/port.log" 2>&1; then
  echo 'expected occupied port rejection' >&2; exit 1
fi
if [[ -s "$TEST_PSQL_LOG" ]]; then
  echo 'runner touched PostgreSQL despite an occupied app port' >&2; exit 1
fi
echo 'scheduler runner safety checks passed'
