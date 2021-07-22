.PHONY: default
default:
	CGO_ENABLED=0 go build -o ./bin/tarian-server ./cmd/tarian-server/
	CGO_ENABLED=0 go build -o ./bin/tarian-cluster-agent ./cmd/tarian-cluster-agent/
	CGO_ENABLED=0 go build -o ./bin/tarian-pod-agent ./cmd/tarian-pod-agent/

proto:
	protoc --experimental_allow_proto3_optional -I=./pkg --go_out=./pkg --go-grpc_out=./pkg --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./pkg/tarianpb/types.proto
	protoc --experimental_allow_proto3_optional -I=./pkg --go_out=./pkg --go-grpc_out=./pkg --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./pkg/tarianpb/api.proto

test:
	go test -v ./pkg/...

lint:
	revive -formatter stylish -config .revive.toml ./pkg/...

