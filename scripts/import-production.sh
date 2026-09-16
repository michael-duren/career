#!/usr/bin/env bash
# Item 7.2: dry-run and real transactional import into the live career-strategy PostgreSQL.
# Reads DATABASE_URL from the deployed Secret so no password is retyped or echoed.
set -euo pipefail
set +x
cd "$(dirname "$0")/.."
export KUBECONFIG="${KUBECONFIG:-/home/mduren/Code/home-infra/infra/ansible/kubeconfig-homelab}"
ns=career-strategy
phase="${1:-help}"
file="${2:-.migration-private/production-export/content.json}"

guard() {
  [[ "$(kubectl config current-context)" == default ]] || { echo "Expected context default" >&2; exit 1; }
  [[ "$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')" == https://192.168.20.3:6443 ]] || { echo "Unexpected API server" >&2; exit 1; }
}

manifest_field() {
  node --input-type=module -e '
    let raw=""; for await (const c of process.stdin) raw+=c;
    const m = JSON.parse(raw).entries.find(e => e.key === "content");
    if (!m) throw new Error("content entry missing from manifest");
    process.stdout.write(String(m["'"$1"'"]));
  ' < .migration-private/production-export/manifest.json
}

run_import() {
  local dryflag="$1"
  [[ -f "$file" ]] || { echo "Export file not found: $file (run scripts/export-netlify-workspace.mjs first)" >&2; exit 1; }
  [[ -f .migration-private/production-export/manifest.json ]] || { echo "manifest.json missing; rerun the export script" >&2; exit 1; }
  guard
  go build -o /tmp/career-import ./cmd/api
  local url revision
  url="$(kubectl -n "$ns" get secret career-secrets -o jsonpath='{.data.DATABASE_URL}' | base64 -d)"
  revision="$(manifest_field revision)"
  DATABASE_URL="$url" /tmp/career-import import \
    --file="$file" \
    --source-store=personal-workspace \
    --source-key=content \
    --source-revision="$revision" \
    $dryflag
}

case "$phase" in
  dry-run)
    run_import --dry-run
    echo "Dry run only: nothing committed. Review the counts above against docs/migration/current-behavior.md before running 'commit'."
    ;;
  commit)
    run_import ""
    echo "Committed. Verify live counts via the app, then consider running rebuild-projections if any board/dashboard view looks stale."
    ;;
  rebuild-projections)
    guard
    go build -o /tmp/career-import ./cmd/api
    url="$(kubectl -n "$ns" get secret career-secrets -o jsonpath='{.data.DATABASE_URL}' | base64 -d)"
    DATABASE_URL="$url" /tmp/career-import rebuild-projections
    ;;
  *)
    echo "Usage: $0 {dry-run|commit|rebuild-projections} [export-file]"
    exit 1
    ;;
esac
