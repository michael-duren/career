#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
image="career-strategy:$(git rev-parse HEAD)"
name="career-step6-check-$$"
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
# A deliberately unavailable database tests the real production binary at startup.
docker run -d --name "$name" -p 127.0.0.1:18089:8080 --read-only --cap-drop=ALL \
  -e DATABASE_URL='postgres://unused:unused@127.0.0.1:1/unused?sslmode=disable&connect_timeout=1' \
  -e AUTH_USERNAME=check \
  -e 'AUTH_PASSWORD_HASH=$2a$04$abcdefghijklmnopqrstuuQzByhP6sxYhbrLqM7T7/kOvP5fRXDVK' \
  -e JWT_SECRET=disposable-container-test-secret-32-characters \
  -e PUBLIC_ORIGIN=https://career.michaelduren.com "$image" >/dev/null
for attempt in {1..30}; do
  if curl -fsS http://127.0.0.1:18089/healthz >/dev/null; then break; fi
  sleep 1
done
curl -fsS http://127.0.0.1:18089/healthz >/dev/null
[[ "$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:18089/readyz)" == 503 ]]
sleep 15
curl -fsS http://127.0.0.1:18089/healthz >/dev/null
docker stop --time=10 "$name" >/dev/null
[[ "$(docker inspect -f '{{.State.ExitCode}}' "$name")" == 0 ]]
echo "PASS: container alive with unavailable DB, readiness=503, SIGTERM exit=0"

