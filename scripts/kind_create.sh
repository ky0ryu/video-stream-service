# run kind thru docker
docker run -d --restart=always \
  -p "127.0.0.1:5001:5000" \
  --network bridge \
  -v kind-registry-data:/var/lib/registry \
  --name kind-registry \
  registry:3

kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/secret.yaml
