## Development Setup

### Pre-requisites

- [Kind](https://kind.sigs.k8s.io/)
- [Go 1.16+](https://golang.org/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)

### Setup


Go to the root directory of this project.

1. Start local registry to be used with kind

```bash
./dev/run-kind-registry.sh
```

2. Build local images

```bash
make local-images
```

3. Create a kind cluster

```bash
make create-kind-cluster
```

4. Apply tarian-k8s manifests

```bash
make deploy
```

Once the pods are running (`kubectl get pods -n tarian-system`),

5. Run DB migration:

```bash
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server db migrate
```

6. Install seed data:

```bash
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server dev seed-data
```

To test that it's working:

7. Run pod:

```bash
kubectl run nginx --image=nginx --annotations=pod-agent.k8s.tarian.dev/inject=true
```

There should be a container injected in nginx pod.

7. Add constraint for that pod:

```bash
./bin/tarianctl --server-address=localhost:31051 add constraint --namespace default --match-labels run=nginx --allowed-processes=pause,tarian-pod-agent,nginx
```

8. Test the violation event

```bash
kubectl exec -ti nginx -c nginx -- sleep 10
```

See there are violation events shortly:

```bash
./bin/tarianctl --server-address=localhost:31051 get events
```
