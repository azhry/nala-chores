#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KUBECTL_BIN="${KUBECTL:-}"
if [[ -z "${KUBECTL_BIN}" ]]; then
  KUBECTL_BIN="$(command -v kubectl || true)"
fi

DOMAIN_HOST="${DOMAIN_HOST:-chores.nalanirvana.com}"
TUNNEL_TOKEN="${CLOUDFLARE_TUNNEL_TOKEN:-${TUNNEL_TOKEN:-}}"

if [[ -z "${KUBECTL_BIN}" || ! -x "${KUBECTL_BIN}" ]]; then
  echo "kubectl is required but was not found in PATH." >&2
  exit 1
fi

if ! command -v minikube >/dev/null 2>&1; then
  echo "minikube is required but was not found in PATH." >&2
  exit 1
fi

if [[ -z "${TUNNEL_TOKEN}" ]]; then
  cat >&2 <<EOF
Set CLOUDFLARE_TUNNEL_TOKEN to the token from Cloudflare Zero Trust > Networks > Tunnels.

The tunnel's published application route should be:
  hostname: ${DOMAIN_HOST}
  service:  http://ingress-nginx-controller.ingress-nginx.svc.cluster.local:80

Optional second route for people who type the www form:
  hostname: www.${DOMAIN_HOST}
  service:  http://ingress-nginx-controller.ingress-nginx.svc.cluster.local:80

Then run:
  CLOUDFLARE_TUNNEL_TOKEN='...' $0
EOF
  exit 1
fi

echo "Ensuring minikube ingress addon is enabled..."
minikube addons enable ingress >/dev/null

echo "Applying Nala Chores public ingress for ${DOMAIN_HOST}..."
"${KUBECTL_BIN}" apply -f "${ROOT_DIR}/deploy/minikube/ingress-public.yaml"

echo "Creating/updating Cloudflare tunnel token secret..."
"${KUBECTL_BIN}" create namespace cloudflare --dry-run=client -o yaml | "${KUBECTL_BIN}" apply -f -
"${KUBECTL_BIN}" -n cloudflare create secret generic cloudflared-token \
  --from-literal=TUNNEL_TOKEN="${TUNNEL_TOKEN}" \
  --dry-run=client -o yaml | "${KUBECTL_BIN}" apply -f -

echo "Deploying cloudflared inside minikube..."
"${KUBECTL_BIN}" apply -f "${ROOT_DIR}/deploy/minikube/cloudflared.yaml"
"${KUBECTL_BIN}" -n cloudflare rollout status deploy/cloudflared --timeout=120s

echo
echo "Done. Open https://${DOMAIN_HOST} after the Cloudflare published application route is saved."
echo "If needed, also publish https://www.${DOMAIN_HOST} to the same service."
