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

Test a scenario:

5. Run pod:

```bash
kubectl run nginx --image=nginx --annotations=pod-agent.k8s.tarian.dev/threat-scan=true
```

There should be a container injected in nginx pod.

6. Add constraint for that pod:

```bash
./bin/tarianctl --server-address=localhost:31051 add constraint --name=nginx --namespace default --match-labels run=nginx --allowed-processes=pause,tarian-pod-agent,nginx
```

7. Test the violation event

```bash
kubectl exec -ti nginx -c nginx -- sleep 10
```

See there are violation events shortly:

```bash
./bin/tarianctl --server-address=localhost:31051 get events
```


## Understanding Tarian components

### tarian-server

`tarian-server` is the central component that stores configurations such as constraints, actions and event logs. This allows us to use a central tarian-server for 
multiple clusters. For example, we want to use tarian in the staging and the production cluster. If tarian registers the known / detected processes in the staging cluster - it can detect the same processes in the production cluster without additional configuration.

### tarian-cluster-agent

`tarian-cluster-agent` is the component that's installed in each cluster and syncs the configurations from `tarian-server`, coordinates with pod-agents, and executes actions.

### tarian-pod-agent

`tarian-pod-agent` is a sidecar container that's injected to pods by `tarian-cluster-agent`. The pod agent periodically scans for threats in the main container, and reports to the `tarian-cluster-agent`.

### tarianctl

`tarianctl` is the CLI application that users can use to interact with the `tarian-server`.


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
