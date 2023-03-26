# Contributing

Tarian welcomes and accepts contributions via GitHub pull requests.

## Pre-requisites

- Kubernetes cluster: minikube or kind
- [Go 1.19+](https://golang.org/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)
- Docker for developing with local cluster

### Kind

```bash
make create-kind-cluster
```

### Minikube

```bash
make create-minikube-cluster
```

## Setup

1. Clone submodules

```bash
git submodule update --init --recursive
```

2. Prepare build tools

```bash
sudo apt update && sudo apt install make unzip pkg-config libelf-dev clang gcc linux-tools-common linux-tools-common linux-tools-generic llvm
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
make bin/protoc bin/goreleaser bin/kustomize
```

Verify if bpftool is working. If it requires to install the packages for your specific kernel, it will recommend the package name.

```bash
bpftool
```

If it's working, the command will print:

```
Usage: /usr/lib/linux-tools/5.15.0-40-generic/bpftool [OPTIONS] OBJECT { COMMAND | help }
       /usr/lib/linux-tools/5.15.0-40-generic/bpftool batch file FILE
       /usr/lib/linux-tools/5.15.0-40-generic/bpftool version

       OBJECT := { prog | map | link | cgroup | perf | net | feature | btf | gen | struct_ops | iter }
       OPTIONS := { {-j|--json} [{-p|--pretty}] | {-d|--debug} |
                    {-V|--version} }
```

3. Build local images

```bash
make local-images
```

4. Apply tarian-k8s manifests

```bash
make deploy
```

5. Wait for all the pods to be ready

```bash
kubectl wait --for=condition=ready pod --all -n tarian-system
```

6. Run DB migration:

```bash
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server dgraph apply-schema
```

## See if it's working

1. Run a pod:

```bash
kubectl run nginx --image=nginx --annotations=pod-agent.k8s.tarian.dev/threat-scan=true
kubectl wait --for=condition=ready pod nginx
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

### tarian-node-agent

`tarian-node-agent` is a daemonset that runs on each node, detecting and reporting unknown processes to the `tarian-cluster-agent`.

### tarian-pod-agent

`tarian-pod-agent` is a sidecar container that's injected to pods by `tarian-cluster-agent`. The pod agent periodically detect unexpected changes to the registered files and reports to the `tarian-cluster-agent`.

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
