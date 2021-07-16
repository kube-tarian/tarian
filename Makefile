.PHONY: default
default:
	CGO_ENABLED=0 go build -a -o ./bin/tarian-server ./cmd/tarian-server/
	CGO_ENABLED=0 go build -a -o ./bin/tarian-cluster-agent ./cmd/tarian-cluster-agent/

proto:
	protoc -I=./pkg --go_out=./pkg --go-grpc_out=./pkg --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./pkg/tarianpb/types.proto
	protoc -I=./pkg --go_out=./pkg --go-grpc_out=./pkg --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./pkg/tarianpb/api.proto

