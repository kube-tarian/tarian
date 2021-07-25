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
