#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="transcoder"

echo "==> Creating kind cluster (if it doesn't already exist)"
if ! kind get clusters | grep -q "^${CLUSTER_NAME}\$"; then
  kind create cluster --config kind-cluster.yaml
else
  echo "cluster '${CLUSTER_NAME}' already exists, skipping"
fi

echo "==> Installing nginx-ingress (kind-specific manifest)"
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
echo "==> Waiting for ingress controller to be ready (can take a minute)"
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s

echo "==> Building images"
docker build -t video-api:dev -f Dockerfile.api ..
docker build -t video-worker:dev -f Dockerfile.worker ..

echo "==> Loading images into kind (kind can't see your local Docker cache otherwise)"
kind load docker-image video-api:dev --name "${CLUSTER_NAME}"
kind load docker-image video-worker:dev --name "${CLUSTER_NAME}"

echo "==> Applying manifests"
kubectl apply -f manifests/

echo "==> Waiting for api and worker rollouts"
kubectl -n transcoder rollout status deployment/postgres --timeout=90s
kubectl -n transcoder rollout status deployment/redis --timeout=90s
kubectl -n transcoder rollout status deployment/api --timeout=90s
kubectl -n transcoder rollout status deployment/worker --timeout=90s

echo "==> Done. API should be reachable at http://localhost:8080"
