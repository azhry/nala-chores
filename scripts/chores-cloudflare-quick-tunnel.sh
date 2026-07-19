#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KUBECTL_BIN="${KUBECTL:-}"
if [[ -z "${KUBECTL_BIN}" ]]; then
  KUBECTL_BIN="$(command -v kubectl || true)"
fi
LOCAL_PORT="${LOCAL_PORT:-8088}"

if [[ -z "${KUBECTL_BIN}" || ! -x "${KUBECTL_BIN}" ]]; then
  echo "kubectl is required but was not found in PATH." >&2
  exit 1
fi

if ! command -v minikube >/dev/null 2>&1; then
  echo "minikube is required but was not found in PATH." >&2
  exit 1
fi

if ! command -v cloudflared >/dev/null 2>&1; then
  echo "cloudflared is required but was not found in PATH." >&2
  exit 1
fi

echo "Ensuring minikube ingress addon is enabled..."
minikube addons enable ingress >/dev/null

echo "Applying hostless ingress for Cloudflare Quick Tunnel..."
"${KUBECTL_BIN}" apply -f "${ROOT_DIR}/deploy/minikube/ingress-cloudflare-quick.yaml"

cleanup() {
  if [[ -n "${PORT_FORWARD_PID:-}" ]]; then
    kill "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

echo "Forwarding local port ${LOCAL_PORT} to the minikube ingress controller..."
"${KUBECTL_BIN}" -n ingress-nginx port-forward svc/ingress-nginx-controller "${LOCAL_PORT}:80" >/tmp/nala-chores-ingress-port-forward.log 2>&1 &
PORT_FORWARD_PID="$!"
sleep 2

if ! kill -0 "${PORT_FORWARD_PID}" >/dev/null 2>&1; then
  echo "port-forward failed:" >&2
  cat /tmp/nala-chores-ingress-port-forward.log >&2
  exit 1
fi

echo
echo "Starting Cloudflare Quick Tunnel for Nala Chores."
echo "Cloudflare will print a temporary https://*.trycloudflare.com URL below."
echo "Leave this process running while you use the URL. Press Ctrl-C to stop."
echo
cloudflared tunnel --url "http://127.0.0.1:${LOCAL_PORT}"
