#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

HOME_INFRA="${HOME_INFRA:-$HOME/Code/home-infra}"
manifests="${CAREER_MANIFESTS:-$HOME_INFRA/k8s/apps/career-strategy}"
export KUBECONFIG="${KUBECONFIG:-$HOME_INFRA/ansible/kubeconfig-homelab}"
k() {
  [[ -f "$KUBECONFIG" ]] || { echo "Kubeconfig not found: $KUBECONFIG" >&2; return 1; }
  kubectl --namespace=career-strategy --request-timeout=15s "$@"
}

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
    exit
    ;;
  rollback)
    k rollout undo deployment/career
    k rollout status deployment/career --timeout=180s
    exit
    ;;
  build|deploy) ;;
  *) echo "Usage: $0 [deploy|build|check|rollback]" >&2; exit 1 ;;
esac

# Every build gets a fresh tag, including builds with uncommitted changes.
image="career-strategy:$(git rev-parse --short HEAD)-$(date -u +%Y%m%d%H%M%S)"
if [[ "${1:-deploy}" == deploy ]]; then
  for file in namespace configmap service ingress deployment migration-job; do
    [[ -f "$manifests/$file.yaml" ]] || { echo "Missing $manifests/$file.yaml" >&2; exit 1; }
  done
  k get secret career-secrets >/dev/null
  # Images must be present on every schedulable node matching the manifests.
  node_ips="$(k get nodes -l kubernetes.io/os=linux,kubernetes.io/arch=amd64 \
    -o go-template='{{range .items}}{{if not .spec.unschedulable}}{{range .status.addresses}}{{if eq .type "InternalIP"}}{{.address}}{{"\n"}}{{end}}{{end}}{{end}}{{end}}')"
  [[ -n "$node_ips" ]] || { echo "No schedulable amd64 nodes found" >&2; exit 1; }
fi

docker build --platform=linux/amd64 -t "$image" .
[[ "${1:-deploy}" != build ]] || { echo "Built $image"; exit; }

while IFS= read -r ip; do
  echo "Importing $image on $ip"
  docker save "$image" | ssh -o BatchMode=yes -o ConnectTimeout=10 "ops@$ip" sudo -n k3s ctr images import -
done <<< "$node_ips"

# Override the app image whether home-infra has a release tag or a placeholder.
render() { sed -E "s|^([[:space:]]*image:).*career-strategy:.*$|\1 $image|" "$manifests/$1.yaml"; }
k apply -f "$manifests/namespace.yaml" -f "$manifests/configmap.yaml"
job="$(render migration-job | k create -f - -o name)"
if ! k wait --for=condition=complete "$job" --timeout=180s; then
  echo "Migration failed ($job); deployment unchanged." >&2
  exit 1
fi
render deployment | k apply -f -
k apply -f "$manifests/service.yaml" -f "$manifests/ingress.yaml"
k rollout status deployment/career --timeout=180s
echo "Deployed $image"
