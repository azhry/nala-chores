#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOSTNAME="chores.nala.local"
HOSTS_LINE="127.0.0.1 ${HOSTNAME}"
KUBECTL_BIN="${KUBECTL:-}"
if [[ -z "${KUBECTL_BIN}" ]]; then
  KUBECTL_BIN="$(command -v kubectl || true)"
fi
KUBECONFIG_PATH="${KUBECONFIG:-${HOME}/.kube/config}"

if ! command -v minikube >/dev/null 2>&1; then
  echo "minikube is required but was not found in PATH." >&2
  exit 1
fi

if [[ -z "${KUBECTL_BIN}" || ! -x "${KUBECTL_BIN}" ]]; then
  echo "kubectl is required but was not found in PATH." >&2
  exit 1
fi

echo "Ensuring minikube ingress addon is enabled..."
minikube addons enable ingress >/dev/null

echo "Applying Nala Chores ingress..."
"${KUBECTL_BIN}" apply -f "${ROOT_DIR}/deploy/minikube/ingress.yaml"

if ! grep -qE "^[[:space:]]*127\\.0\\.0\\.1[[:space:]].*\\b${HOSTNAME}\\b" /etc/hosts; then
  echo "Adding ${HOSTNAME} to /etc/hosts. macOS may ask for your password."
  printf "\n%s\n" "${HOSTS_LINE}" | sudo tee -a /etc/hosts >/dev/null
fi

echo
echo "Starting local reverse proxy:"
echo "  http://${HOSTNAME}"
echo
echo "Leave this process running while you use the URL. Press Ctrl-C to stop."
sudo -E env "KUBECONFIG=${KUBECONFIG_PATH}" "${KUBECTL_BIN}" \
  -n ingress-nginx port-forward svc/ingress-nginx-controller 80:80
