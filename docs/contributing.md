# Contributing

Tarian welcomes and accepts contributions via GitHub pull requests.

## Pre-requisites

- Kubernetes cluster: minikube or kind
- [Go 1.19+](https://golang.org/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)
- Docker for developing with local cluster
- [Helm](https://helm.sh/)
- You should be root user to run tarian

## Setup

1. Clone the repo with submodules

```bash
git clone --recurse-submodules https://github.com/kube-tarian/tarian.git
```

2. Kind

This step will create a kind cluster with a local image registry hosted on localhost:5000
```bash
make create-kind-cluster
```
OR

Minikube

```bash
make create-minikube-cluster
```

3. Prepare build tools

```bash
sudo apt update && sudo apt install make unzip pkg-config libelf-dev clang gcc linux-tools-common linux-tools-common linux-tools-generic
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

In case if you see warning:
```
WARNING: bpftool not found for kernel 5.15.0-87
 
  You may need to install the following packages for this specific kernel:
    linux-tools-5.15.0-87-generic
    linux-cloud-tools-5.15.0-87-generic
 
  You may also want to install one of the following packages to keep up to date:
    linux-tools-generic
    linux-cloud-tools-generic
```
Install the packages listed above.

4. Build local images and binaries

```bash
make local-images
```
All the binaries are in `./bin` and images are pushed to local image registry

5. Install tarian using install command
```
export PATH=$PATH:./bin

tarianctl install --charts charts -l debug --agents-values dev/values/agents.yaml --server-values dev/values/server.yaml
```

or

```bash
make deploy
```
If you use make deploy then follow next two steps:

1. Wait for all the pods to be ready

```bash
kubectl wait --for=condition=ready pod --all -n tarian-system
```

2. Once the pods are up then run DB migration:

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
