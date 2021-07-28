.PHONY: default
default:
	CGO_ENABLED=0 go build -o ./bin/tarian-server ./cmd/tarian-server/
	CGO_ENABLED=0 go build -o ./bin/tarian-cluster-agent ./cmd/tarian-cluster-agent/
	CGO_ENABLED=0 go build -o ./bin/tarian-pod-agent ./cmd/tarian-pod-agent/
	CGO_ENABLED=0 go build -o ./bin/tarianctl ./cmd/tarianctl/

proto:
	protoc --experimental_allow_proto3_optional -I=./pkg --go_out=./pkg --go-grpc_out=./pkg --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./pkg/tarianpb/types.proto
	protoc --experimental_allow_proto3_optional -I=./pkg --go_out=./pkg --go-grpc_out=./pkg --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./pkg/tarianpb/api.proto

unit-test:
	go test -v ./pkg/...

e2e-test:
	go test -v ./test/e2e/...

lint:
	revive -formatter stylish -config .revive.toml ./pkg/...

local-images:
	docker build -f Dockerfile-server -t localhost:5000/tarian-server . && docker push localhost:5000/tarian-server
	docker build -f Dockerfile-cluster-agent -t localhost:5000/tarian-cluster-agent . && docker push localhost:5000/tarian-cluster-agent
	docker build -f Dockerfile-pod-agent -t localhost:5000/tarian-pod-agent . && docker push localhost:5000/tarian-pod-agent
