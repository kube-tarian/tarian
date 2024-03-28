# Image URL to use all building/pushing image targets
IMG ?= localhost:5000/tarian-cluster-agent

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

default: help


##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


BASEDIR = $(abspath ./)
OUTPUT = ./output
ARCH := $(shell uname -m | sed 's/x86_64/amd64/g; s/aarch64/arm64/g')

CC = gcc
GO = go
CFLAGS = -g -O2 -Wall -fpie
LDFLAGS =

CGO_CFLAGS_STATIC = "-I$(abspath $(OUTPUT)) -Wno-unknown-attributes"
CGO_LDFLAGS_STATIC = "-lelf -lz $(LIBBPF_OBJ)"
CGO_EXTLDFLAGS_STATIC = '-w -extldflags "-static"'
CGO_CFGLAGS_DYN = "-I. -I/usr/include/"
CGO_LDFLAGS_DYN = "-lelf -lz -lbpf"

BTFFILE = /sys/kernel/btf/vmlinux
BPFTOOL = $(shell which bpftool || /bin/false)
VMLINUXH = $(OUTPUT)/vmlinux.h

# output

$(OUTPUT):
	mkdir -p $(OUTPUT)

# vmlinux header file

.PHONY: vmlinuxh
vmlinuxh: $(VMLINUXH)

$(VMLINUXH): $(OUTPUT)
	@if [ ! -f $(BTFFILE) ]; then \
		echo "ERROR: kernel does not seem to support BTF"; \
		exit 1; \
	fi
	@if [ ! -f $(VMLINUXH) ]; then \
		if [ ! $(BPFTOOL) ]; then \
			echo "ERROR: could not find bpftool"; \
			exit 1; \
		fi; \
		echo "INFO: generating $(VMLINUXH) from $(BTFFILE)"; \
		$(BPFTOOL) btf dump file $(BTFFILE) format c > $(VMLINUXH); \
	fi
	
##@ Development

generate: bin/controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/clusteragent/..." -w

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	CGO_CFLAGS=$(CGO_CFLAGS_STATIC) CGO_LDFLAGS=$(CGO_LDFLAGS_STATIC) go vet ./...

build: bin/goreleaser generate proto ## Build binaries and copy to ./bin/
	./bin/goreleaser build --single-target --snapshot --rm-dist --single-target
	cp dist/*/tarian* ./bin/

proto: bin/protoc
	$(PROTOC) --experimental_allow_proto3_optional=true -I=./.local/include -I=./pkg --go_out=./pkg --go-grpc_out=./pkg --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./pkg/tarianpb/types.proto
	$(PROTOC) --experimental_allow_proto3_optional=true -I=./.local/include -I=./pkg --go_out=./pkg --go-grpc_out=./pkg --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./pkg/tarianpb/api.proto

lint: ## Run golangci-lint against code.
	docker run --rm -v $(BASEDIR):/app -w /app golangci/golangci-lint:v1.54.2 golangci-lint run -v --config=.golangci.yml

local-images: build
	docker build -f Dockerfile-server -t localhost:5000/tarian-server dist/tarian-server_linux_amd64/ && docker push localhost:5000/tarian-server
	docker build -f Dockerfile-cluster-agent -t localhost:5000/tarian-cluster-agent dist/tarian-cluster-agent_linux_amd64/ && docker push localhost:5000/tarian-cluster-agent
	docker build -f Dockerfile-pod-agent -t localhost:5000/tarian-pod-agent dist/tarian-pod-agent_linux_amd64/ && docker push localhost:5000/tarian-pod-agent
	docker build -f Dockerfile-node-agent -t localhost:5000/tarian-node-agent dist/tarian-node-agent_linux_amd64/ && docker push localhost:5000/tarian-node-agent
	docker build -f Dockerfile-tarianctl -t localhost:5000/tarianctl dist/tarianctl_linux_amd64/ && docker push localhost:5000/tarianctl

push-local-images:
	docker push localhost:5000/tarian-server
	docker push localhost:5000/tarian-cluster-agent
	docker push localhost:5000/tarian-pod-agent
	docker push localhost:5000/tarian-node-agent

unit-test:
	CGO_CFLAGS=$(CGO_CFLAGS_STATIC) CGO_LDFLAGS=$(CGO_LDFLAGS_STATIC) go test -v -race -count=1 -coverprofile=coverage.xml -covermode=atomic ./pkg/... ./cmd/...

e2e-test:
	CGO_CFLAGS=$(CGO_CFLAGS_STATIC) CGO_LDFLAGS=$(CGO_LDFLAGS_STATIC) go test -v -race -count=1 ./test/e2e/...

k8s-test:
	./test/k8s/test.sh

coverage: unit-test
	go tool cover -html=coverage.xml
	
manifests: bin/controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) webhook paths="./pkg/clusteragent/..." output:webhook:artifacts:config=dev/config/webhook

create-kind-cluster:
	./dev/run-kind-registry.sh
	kind create cluster --config=dev/cluster-config.yaml --name tarian

delete-kind-cluster:
	kind delete cluster --name tarian

create-minikube-cluster:
	minikube start --driver=virtualbox --insecure-registry "10.0.0.0/8"
	minikube addons enable registry
	bash -c 'docker start minikube-registry || docker run --name=minikube-registry --rm -ti -d --network=host alpine /bin/ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:`minikube ip`:5000"'

delete-minikube-cluster:
	docker rm minikube-registry -f
	minikube delete

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
controller-test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out


##@ Deployment

deploy: manifests bin/kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd dev/config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build dev/config/default | kubectl apply -f -
	helm repo add nats https://nats-io.github.io/k8s/helm/charts/
	helm upgrade -i nats nats/nats -n tarian-system -f ./dev/nats-helm-values.yaml --version 0.19.16

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build dev/config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
bin/controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.3)

KUSTOMIZE = $(shell pwd)/bin/kustomize
bin/kustomize: ## Download kustomize locally if necessary.
	curl -o kustomize_v4.5.7_linux_amd64.tar.gz -L0 "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v4.5.7/kustomize_v4.5.7_linux_amd64.tar.gz"
	mkdir -p bin
	tar -C ./bin/ -xvf kustomize_v4.5.7_linux_amd64.tar.gz
	rm -f kustomize_v4.5.7_linux_amd64.tar.gz

PROTOC = $(shell pwd)/bin/protoc
PROTOC_ZIP = protoc-3.17.3-linux-x86_64.zip
bin/protoc:
	curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v3.17.3/$(PROTOC_ZIP)" 
	unzip -o $(PROTOC_ZIP) -d ./ bin/protoc
	unzip -o $(PROTOC_ZIP) -d ./.local 'include/*'
	rm -f $(PROTOC_ZIP)
	chmod +x ./bin/protoc

GORELEASER = $(shell pwd)/bin/goreleaser
bin/goreleaser:
	curl -LO "https://github.com/goreleaser/goreleaser/releases/download/v0.174.2/goreleaser_Linux_x86_64.tar.gz"
	mkdir -p bin
	tar -C ./bin/ -xvf goreleaser_Linux_x86_64.tar.gz goreleaser
	rm -f goreleaser_Linux_x86_64.tar.gz

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
