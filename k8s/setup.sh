#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(realpath "${SCRIPT_DIR}/..")"
K8S_DIR="${SCRIPT_DIR}/manifests"
CLUSTER_NAME="transcoder"

echo "==> Creating kind cluster (if it doesn't already exist)"
if ! kind get clusters | grep -q "^${CLUSTER_NAME}\$"; then
  kind create cluster --config "${SCRIPT_DIR}/kind-cluster.yaml"

  # Ensure API server is fully responsive before proceeding
  echo "==> Waiting for Kubernetes API server to become ready..."
  until kubectl cluster-info &>/dev/null; do
    sleep 2
  done
else
  echo "cluster '${CLUSTER_NAME}' already exists, skipping"
fi

echo "==> Installing nginx-ingress (pinned version v1.11.2)"
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.2/deploy/static/provider/kind/deploy.yaml

echo "==> Waiting for ingress-nginx deployment rollout..."
kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx --timeout=120s

echo "==> Building images"
docker build -t video-api:dev -f "${ROOT_DIR}/Dockerfile.api" "${ROOT_DIR}"
docker build -t video-worker:dev -f "${ROOT_DIR}/Dockerfile.worker" "${ROOT_DIR}"

echo "==> Loading images into kind"
kind load docker-image video-api:dev --name "${CLUSTER_NAME}"
kind load docker-image video-worker:dev --name "${CLUSTER_NAME}"

echo "==> Applying manifests"
kubectl apply -f "${K8S_DIR}/namespace.yaml"
kubectl apply -f "${K8S_DIR}"

echo "==> Waiting for postgres rollout"
kubectl -n transcoder rollout status deployment/postgres --timeout=120s

echo "==> Waiting for redis rollout"
kubectl -n transcoder rollout status deployment/redis --timeout=120s

echo "==> Running DB migrations (Job: migrate)"
kubectl -n transcoder create configmap migrations \
  --from-file="${ROOT_DIR}/dbMigration/" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n transcoder delete job migrate --ignore-not-found
kubectl apply -f "${K8S_DIR}/migrate.yaml"

if ! kubectl -n transcoder wait --for=condition=complete job/migrate --timeout=180s; then
  echo "ERROR: migration failed. Logs:"
  kubectl -n transcoder logs job/migrate --tail=50
  exit 1
fi

echo "==> Waiting for api rollout"
kubectl -n transcoder rollout status deployment/api --timeout=120s

echo "==> Waiting for worker rollout"
kubectl -n transcoder rollout status deployment/worker --timeout=120s

echo "==> Done. API should be reachable at http://localhost:8080"
