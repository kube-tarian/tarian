# Contributing

Tarian welcomes and accepts contributions via GitHub pull requests.

## Pre-requisites

- [A kubernetes cluster that is able to run falco](https://falco.org/docs/getting-started/third-party/learning/) 
  - minikube
  - or kind, with falco installed on the host
  - or a managed (cloud) kubernetes cluster with ability to run falco
- [Go 1.17+](https://golang.org/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)
- Docker for developing with local cluster

If you want to go with local cluster, we have makefile targets to help you:

### Kind

After installing falco on the host (https://falco.org/docs/getting-started/installation/), run the following command:

```bash
make create-kind-cluster
```

### Minikube

```bash
make create-minikube-cluster
```


## Setup

1. Prepare build tools

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
make bin/protoc bin/goreleaser kustomize
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

## See if it's working

1. Run a pod:

```bash
kubectl run nginx --image=nginx --annotations=pod-agent.k8s.tarian.dev/threat-scan=true
```

There should be a container injected in nginx pod.

2. Add constraint for that pod:

```bash
./bin/tarianctl --server-address=localhost:31051 add constraint --name=nginx --namespace default --match-labels run=nginx --allowed-processes=pause,tarian-pod-agent,nginx
```

3. Test the violation event

```bash
kubectl exec -ti nginx -c nginx -- sleep 10
```

See that there are violation events:

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
