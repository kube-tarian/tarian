# Contributing

We welcome and accepts contributions via GitHub pull requests.

## Run locally

### Pre-requisites

- [Kind](https://kind.sigs.k8s.io/)
- [Go 1.17+](https://golang.org/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)

### Setup


Go to the root directory of this project.

1. Create a kind cluster

```bash
make create-kind-cluster
```

2. Build local images

```bash
make local-images
```

3. Apply tarian-k8s manifests

```bash
make deploy
```

Once the pods are running (`kubectl get pods -n tarian-system`),

4. Run DB migration:

```bash
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server db migrate
```

5. Install seed data:

```bash
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server dev seed-data
```

To test that it's working:

6. Run pod:

```bash
kubectl run nginx --image=nginx --annotations=pod-agent.k8s.tarian.dev/threat-scan=true
```

There should be a container injected in nginx pod.

7. Add constraint for that pod:

```bash
./bin/tarianctl --server-address=localhost:31051 add constraint --name=nginx --namespace default --match-labels run=nginx --allowed-processes=pause,tarian-pod-agent,nginx
```

8. Test the violation event

```bash
kubectl exec -ti nginx -c nginx -- sleep 10
```

See there are violation events shortly:

```bash
./bin/tarianctl --server-address=localhost:31051 get events
```

## Understanding Tarian components

### tarian-server

### tarian-cluster-agent

### tarian-pod-agent

### tarianctl

## How to run the test suite

### Test suite without Kubernetes

```bash
docker-compose up -d
make unit-test
make e2e-test
```

### Test suite with Kubernetes

```bash
make k8s-test
```

## Acceptance policy

## Commit Message format
