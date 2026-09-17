#!/usr/bin/env bash
set -euo pipefail
set +x
cd "$(dirname "$0")/.."
export KUBECONFIG="${KUBECONFIG:-/home/mduren/Code/home-infra/infra/ansible/kubeconfig-homelab}"
phase="${1:-help}"
ns=career-strategy
# Manifests live in home-infra (k8s/apps/career-strategy); this repo owns the
# build, the image import and the secrets.
HOME_INFRA="${HOME_INFRA:-$HOME/Code/home-infra}"
manifests="${CAREER_MANIFESTS:-$HOME_INFRA/k8s/apps/career-strategy}"
k() { kubectl --context=default --namespace="$ns" --request-timeout=15s "$@"; }
guard() {
  [[ -f "$manifests/deployment.yaml" ]] || { echo "Manifests not found in $manifests; set HOME_INFRA or CAREER_MANIFESTS" >&2; exit 1; }
  [[ "$(kubectl config current-context)" == default ]] || { echo "Expected context default" >&2; exit 1; }
  [[ "$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')" == https://192.168.20.3:6443 ]] || { echo "Unexpected API server" >&2; exit 1; }
}
release() {
  [[ -z "$(git status --porcelain --untracked-files=normal)" ]] || { echo "Review and commit this worktree before building/releasing an immutable image." >&2; exit 1; }
  image="career-strategy:$(git rev-parse HEAD)"
}
nodes() {
  k get nodes -o json | node --input-type=module -e '
    let raw=""; for await (const c of process.stdin) raw+=c;
    const nodes=JSON.parse(raw).items.filter(n=>!n.spec.unschedulable &&
      !(n.spec.taints||[]).some(t=>["NoSchedule","NoExecute"].includes(t.effect)) &&
      n.metadata.labels["kubernetes.io/os"]==="linux" &&
      n.metadata.labels["kubernetes.io/arch"]==="amd64");
    if (!nodes.length) throw Error("No eligible amd64 nodes");
    for (const n of nodes) {
      if (!n.status.conditions.some(c=>c.type==="Ready"&&c.status==="True")) throw Error("Eligible node not Ready: "+n.metadata.name);
      const ip=n.status.addresses.find(a=>a.type==="InternalIP")?.address;
      if (!/^192\.168\.20\.\d+$/.test(ip)) throw Error("Review unexpected node IP: "+ip);
      console.log(ip);
    }'
}
render() { sed "s|career-strategy:RELEASE|$image|g" "$manifests/$1"; }
import_image() {
  docker image inspect "$1" >/dev/null
  for ip in "${eligible[@]}"; do
    echo "Importing $1 on $ip"
    docker save "$1" | ssh -o BatchMode=yes -o ConnectTimeout=5 "ops@$ip" sudo -n k3s ctr images import -
  done
}
case "$phase" in
  check)
    npm ci
    npx --no-install astro check
    npm run tc -- --noEmit
    npm test
    npm run build
    go test -race ./...
    go build -o /tmp/career-migration ./cmd/api
    node tests/go-api.integration.mjs
    echo "PASS: source checks (PostgreSQL race coverage requires check-postgres)"
    ;;
  check-postgres)
    make test-postgres
    ;;
  inspect)
    guard
    k get nodes -o wide
    k get ingress -A
    k get deployment career --ignore-not-found -o wide
    nodes
    ;;
  build)
    release
    if docker image inspect "$image" >/dev/null 2>&1; then
      echo "Image already exists: $image (retained unchanged)"
    else
      docker build --platform=linux/amd64 -t "$image" .
    fi
    docker run --rm --entrypoint /bin/sh "$image" -ec '
      test "$(id -u)" = 65532
      test -x /app/career
      test -f /app/dist/login/index.html
      test ! -e /app/src
      test ! -e /app/.env
      ! command -v node
      ! find /app/dist -type f | grep -E "\.(map|sql)$|/\.env|workspace-v2|seed-v2"
    '
    echo "PASS: $image built; runtime inventory checked"
    ;;
  secrets)
    guard
    k apply -f "$manifests/namespace.yaml"
    umask 077
    secret_dir="$(mktemp -d)"
    trap 'rm -f "$secret_dir/DATABASE_URL" "$secret_dir/AUTH_USERNAME" "$secret_dir/AUTH_PASSWORD_HASH" "$secret_dir/JWT_SECRET"; rmdir "$secret_dir"' EXIT
    read -r -s -p "Step 5 career_strategy database password: " password; echo
    [[ ${#password} -ge 20 ]]
    printf '%s' "$password" | node --input-type=module -e '
      let p=""; for await(const c of process.stdin)p+=c;
      process.stdout.write("postgres://career_strategy:"+encodeURIComponent(p)+"@192.168.20.103:5432/career_strategy?sslmode=disable");
    ' > "$secret_dir/DATABASE_URL"
    unset password
    read -r -p "Browser username: " username
    [[ -n "$username" ]]
    printf '%s' "$username" > "$secret_dir/AUTH_USERNAME"
    read -r -s -p "Browser password (8–72 bytes): " password; echo
    printf '%s' "$password" | node --input-type=module -e '
      import bcrypt from "bcryptjs";
      let p=""; for await(const c of process.stdin)p+=c;
      if(Buffer.byteLength(p)<8||Buffer.byteLength(p)>72) throw Error("Password must be 8–72 bytes");
      process.stdout.write(bcrypt.hashSync(p,12));
    ' > "$secret_dir/AUTH_PASSWORD_HASH"
    unset password
    node -e 'process.stdout.write(require("node:crypto").randomBytes(48).toString("base64url"))' > "$secret_dir/JWT_SECRET"
    k create secret generic career-secrets --from-file="$secret_dir" --dry-run=client -o json |
      k apply --server-side --field-manager=career-secrets -f - >/dev/null
    echo "Secret updated; deploy to restart pods. Existing sessions will expire."
    ;;
  validate|deploy|sync-images)
    guard
    release
    node_list="$(nodes)"
    mapfile -t eligible <<< "$node_list"
    previous="$(k get deployment career --ignore-not-found -o jsonpath='{.spec.template.spec.containers[0].image}')"
    if [[ "$phase" == sync-images ]]; then
      previous="$(k get deployment career -o go-template='{{index .metadata.annotations "career-strategy/previous-image"}}')"
      [[ "$previous" != '<no value>' ]] || previous=''
    fi
    if [[ "$phase" == sync-images || "$phase" == deploy ]]; then
      if [[ -n "$previous" && "$previous" != "$image" ]]; then import_image "$previous"; fi
      import_image "$image"
    fi
    [[ "$phase" != sync-images ]] || exit 0
    if [[ "$phase" == validate ]]; then
      for file in configmap.yaml service.yaml ingress.yaml deployment.yaml; do
        render "$file" | k apply --dry-run=server -f - >/dev/null
      done
      render migration-job.yaml | k create --dry-run=server -f - >/dev/null
      echo "PASS: server manifest validation"
      exit 0
    fi
    k get secret career-secrets >/dev/null
    k apply -f "$manifests/configmap.yaml"
    job="$(render migration-job.yaml | k create -f - -o name)"
    if ! k wait --for=condition=complete "$job" --timeout=180s; then
      echo "Migration failed. Deployment unchanged. Inspect $job securely; do not paste secret-bearing logs." >&2
      exit 1
    fi
    # Save rollback before changing the Deployment, including failed rollouts.
    if [[ -n "$previous" && "$previous" != "$image" ]]; then
      k annotate deployment career "career-strategy/previous-image=$previous" --overwrite
    fi
    render deployment.yaml | k apply -f -
    k apply -f "$manifests/service.yaml" -f "$manifests/ingress.yaml"
    k rollout restart deployment/career
    k rollout status deployment/career --timeout=180s
    echo "PASS: release=$image previous=${previous:-none}; no public DNS changed"
    ;;
  rollback)
    guard
    previous="$(k get deployment career -o go-template='{{index .metadata.annotations "career-strategy/previous-image"}}')"
    [[ "$previous" =~ ^career-strategy:[a-f0-9]{40}$ ]] || { echo "No recorded rollback image"; exit 1; }
    node_list="$(nodes)"; mapfile -t eligible <<< "$node_list"
    import_image "$previous"
    k set image deployment/career "career=$previous"
    k rollout status deployment/career --timeout=180s
    ;;
  tunnel)
    guard
    read -r -p "Reviewed cloudflared image (cloudflare/cloudflared@sha256:...): " tunnel_image
    [[ "$tunnel_image" =~ ^cloudflare/cloudflared@sha256:[a-f0-9]{64}$ ]] || exit 1
    umask 077
    secret_dir="$(mktemp -d)"
    trap 'rm -f "$secret_dir/TUNNEL_TOKEN"; rmdir "$secret_dir"' EXIT
    read -r -s -p "Dedicated career tunnel token: " token; echo
    [[ -n "$token" ]]
    printf '%s' "$token" > "$secret_dir/TUNNEL_TOKEN"
    unset token
    k create secret generic career-tunnel --from-file="$secret_dir" --dry-run=client -o json |
      k apply --server-side --field-manager=career-secrets -f - >/dev/null
    sed "s|CLOUDFLARED_IMAGE|$tunnel_image|" "$manifests/cloudflared.yaml" | k apply -f -
    k rollout restart deployment/career-tunnel
    k rollout status deployment/career-tunnel --timeout=180s
    echo "Tunnel deployed; verify provider routes separately before acceptance."
    ;;
  logs)
    guard
    k logs deployment/career --tail=100
    ;;
  *)
    echo "Usage: $0 {check|check-postgres|inspect|build|secrets|validate|deploy|sync-images|rollback|tunnel|logs}"
    [[ "$phase" == help ]]
    ;;
esac
