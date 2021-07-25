## Development Setup

1. Start local registry to be used with kind

```bash
./run-kind-registry.sh

```

2. Create a kind cluster with this config:

```bash
kind create cluster --config=cluster-config.yaml
```

3. Apply registry config map:

```bash
kubectl apply -f registry-hosting-config-map.yaml
```

4. Apply tarian-k8s manifests:

```bash
kubectl create namespace tarian-system
kubectl apply -f tarian-k8s -n tarian-system
```

Once the pods are running,


5. Run DB migration:

```bash
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server db migrate
```

6. Install seed data:

```bash
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server dev seed-data
```

