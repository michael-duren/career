#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

case "${1:-deploy}" in
  check)
    npm ci
    npx --no-install astro check
    npm run tc -- --noEmit
    npm test
    npm run build
    go test -race ./...
    go build -o /tmp/career-migration ./cmd/api
    node tests/go-api.integration.mjs
    ;;
  build)
    image="career-strategy:$(git rev-parse --short HEAD)-$(date -u +%Y%m%d%H%M%S)"
    docker build --platform=linux/amd64 -t "$image" .
    echo "Built $image"
    ;;
  deploy|rollback)
    echo 'Career deployment and rollback now use home-infra GitOps; see docs/releases.md.' >&2
    echo 'Local apply/import is disabled to protect Argo ownership and UNRELEASED templates.' >&2
    exit 1
    ;;
  *) echo "Usage: $0 [build|check]" >&2; exit 1 ;;
esac
