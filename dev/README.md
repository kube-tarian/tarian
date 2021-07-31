## Development Setup

### Pre-requisites

- [Kind](https://kind.sigs.k8s.io/)
- [Go 1.16+](https://golang.org/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)

### Setup

1. Start local registry to be used with kind

```bash
./run-kind-registry.sh
```

2. Build local images

```bash
cd ..
make build local-images
```

3. Create a kind cluster with this config:

```bash
make create-kind-cluster
```

4. Apply tarian-k8s manifests:

```bash
make deploy
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

